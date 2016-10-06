package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

//A castle has four guards and a watchtower.
//On any set time, only three of the guards are on watch
//When the watchtower spots a dragon, all (and only) the guards on watch should leave
//As the watchtower can't be busy preparing schedules and accepting changes to it whenever guards
//trade watches, they need a way to inform the watchtower of who is currently on watch
func main() {
	rand.Seed(time.Now().UnixNano())

	events := make(EventChannel)
	quit := make(chan struct{})

	dragon := &Dragon{HP: 3}
	castle := NewCastle(4)

	go func(quit chan struct{}) {
		for {
			event := <-events
			switch event := event.(type) {
			case GuardOnWatchEvent:
				log.Printf("Guard %v on duty", event)
			case GuardOffDutyEvent:
				log.Printf("Guard %v off duty", event)
			case CastleUnderAttackEvent:
				log.Println("A dragon attacks your castle")
			case GuardAttacksEvent:
				log.Println("A guard attacks the dragon")
			case DragonDamagedEvent:
				log.Printf("Dragon damaged. Hp left: %v", event.HP)
			case CastleDestroyedEvent:
				log.Println("The dragon destroyed your castle")
				close(quit)
				break
			case DragonDeadEvent:
				log.Println("Your guards defeated the dragon")
				close(quit)
				break
			}
		}
	}(quit)

	castle.AssignGuards(events)

	dragon.Approach(castle, events)

	<-quit
}

type GuardOnWatchEvent int
type GuardOffDutyEvent int
type CastleUnderAttackEvent struct{}
type CastleDestroyedEvent struct{}
type DragonDeadEvent struct{}
type GuardAttacksEvent struct{}
type DragonDamagedEvent struct {
	HP int
}

type EventChannel chan interface{}

type Danger interface {
	TakeDamage(EventChannel)
}

type Adversary interface {
	Spot(Danger, EventChannel)
	Destroy(EventChannel)
}

type Dragon struct {
	sync.RWMutex
	HP int
}

func (dragon *Dragon) TakeDamage(events EventChannel) {
	dragon.Lock()
	defer dragon.Unlock()

	dragon.HP--
	events <- DragonDamagedEvent{HP: dragon.HP}

	if !dragon.isAlive() {
		events <- DragonDeadEvent{}
	}
}

//This function assumes the caller already locked
//the mutex on dragon for reading (or writing)
func (dragon *Dragon) isAlive() bool {
	return dragon.HP > 0
}

func (dragon *Dragon) IsAlive() bool {
	dragon.RLock()
	defer dragon.RUnlock()

	return dragon.isAlive()
}

func (dragon *Dragon) Approach(adversary Adversary, events EventChannel) {
	adversary.Spot(dragon, events)

	go func(events EventChannel) {
		<-time.After(2 * time.Second)
		adversary.Destroy(events)
	}(events)
}

func NewCastle(nGuards int) *Castle {
	castle := &Castle{
		Watchtower: Watchtower{Horn: make(chan Danger), guards: nGuards - 1},
		destroyed:  make(chan struct{}),
	}

	for i := 0; i < nGuards; i++ {
		guard := *NewGuard()
		castle.Guards = append(castle.Guards, guard)
	}

	return castle
}

type Castle struct {
	Guards []Guard
	Watchtower
	destroyed chan struct{}
}

func (castle *Castle) AssignGuards(events EventChannel) {
	offDuty := rand.Intn(len(castle.Guards))

	for i, guard := range castle.Guards {
		if i == offDuty {
			events <- GuardOffDutyEvent(i)
			guard.OffDuty()
		} else {
			events <- GuardOnWatchEvent(i)
			guard.StandWatch(castle.Watchtower.Horn, events)
		}
	}
}

func (castle *Castle) Destroy(events EventChannel) {
	events <- CastleDestroyedEvent{}
}

func NewGuard() *Guard {
	return &Guard{watchDone: make(chan struct{})}
}

type Guard struct {
	watchDone chan struct{}
}

func (guard Guard) StandWatch(Horn chan Danger, events EventChannel) {
	go func() {
		for {
			select {
			case danger := <-Horn:
				guard.Attack(danger, events)
			case <-guard.watchDone:
				return
			}
		}
	}()
}

func (guard Guard) OffDuty() {
	close(guard.watchDone)
	guard.watchDone = make(chan struct{})
}

func (guard Guard) Attack(danger Danger, events EventChannel) {
	events <- GuardAttacksEvent{}
	danger.TakeDamage(events)
}

type Watchtower struct {
	Horn   chan Danger
	guards int
}

func (watchtower Watchtower) Spot(danger Danger, events EventChannel) {
	events <- CastleUnderAttackEvent{}

	for i := 0; i < watchtower.guards; i++ {
		watchtower.Horn <- danger
	}
}
