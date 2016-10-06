package main

import (
	"log"
	"math/rand"
	"time"
)

//A castle has four guards and a watchtower.
//On any set time, only three of the guards are on watch
//When the watchtower spots a dragon, all (and only) the guards on watch should leave
//As the watchtower can't be busy preparing schedules and accepting changes to it whenever guards
//trade watches, they need a way to inform the watchtower of who is currently on watch
func main() {
	rand.Seed(time.Now().UnixNano())
	dragon := &Dragon{HP: 3, dead: make(chan struct{})}
	castle := NewCastle(4)

	castle.AssignGuards()

	log.Println("A dragon attacks your castle")
	dragon.Approach(castle)

	select {
	case <-castle.Destroyed():
		log.Println("The dragon destroyed your castle")
	case <-dragon.Dead():
		log.Println("Your guards defeated the dragon")
	}
}

type Danger interface {
	TakeDamage()
}

type Adversary interface {
	Spot(Danger)
	Destroy()
}

type Dragon struct {
	HP   int
	dead chan struct{}
}

func (dragon *Dragon) TakeDamage() {
	dragon.HP--
	log.Printf("Dragon damaged. Hp left: %v", dragon.HP)

	if !dragon.IsAlive() && dragon.dead != nil {
		close(dragon.dead)
	}
}

func (dragon *Dragon) IsAlive() bool {
	return dragon.HP > 0
}

func (dragon *Dragon) Dead() chan struct{} {
	return dragon.dead
}

func (dragon *Dragon) Approach(adversary Adversary) {
	adversary.Spot(dragon)

	go func() {
		<-time.After(2 * time.Second)
		adversary.Destroy()
	}()
}

func NewCastle(nGuards int) *Castle {
	castle := &Castle{
		Watchtower: Watchtower{Horn: make(chan Danger), guards: nGuards},
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

func (castle Castle) AssignGuards() {
	offDuty := rand.Intn(4)

	for i, guard := range castle.Guards {
		if i == offDuty {
			log.Printf("Guard %v off duty", i)
			guard.OffDuty()
		} else {
			log.Printf("Guard %v on watch", i)
			guard.AssignWatch(castle.Watchtower.Horn)
		}
	}
}

func (castle Castle) Destroy() {
	close(castle.destroyed)
}

func (castle Castle) Destroyed() chan struct{} {
	return castle.destroyed
}

func NewGuard() *Guard {
	return &Guard{watchDone: make(chan struct{})}
}

type Guard struct {
	watchDone chan struct{}
}

func (guard Guard) AssignWatch(Horn chan Danger) {
	go func() {
		for {
			log.Println("Listening for danger")
			select {
			case danger := <-Horn:
				guard.Attack(danger)
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

func (guard Guard) Attack(danger Danger) {
	log.Println("Attacking")
	danger.TakeDamage()
}

type Watchtower struct {
	Horn   chan Danger
	guards int
}

func (watchtower Watchtower) Spot(danger Danger) {
	for i := 0; i < watchtower.guards; i++ {
		watchtower.Horn <- danger
	}
}
