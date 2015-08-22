package gofsm

import (
	"testing"
	"time"

	. "github.com/motain/gocheck"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct{}

var _ = Suite(&S{})

func (s *S) TestEvents(c *C) {
	aut, err := LoadFile("examples/simple.yaml")
	c.Assert(err, Equals, nil)
	dog, ok := aut.Automaton["simple"]
	c.Assert(ok, Equals, true)
	c.Assert(dog.State.Name, Equals, "Hungry")

	// event
	dog.Process("food.meat")
	c.Assert(dog.State.Name, Equals, "Eating")
	c.Assert((<-aut.Actions).Name, Equals, "woof()")
	c.Assert((<-aut.Actions).Name, Equals, "eat('apple')")
	ch := <-aut.Changes
	c.Assert(ch.Old, Equals, "Hungry")
	c.Assert(ch.New, Equals, "Eating")
	c.Assert(ch.Duration < time.Millisecond, Equals, true)

	// event
	dog.Process("food.meat")
	c.Assert(dog.State.Name, Equals, "Full")
	c.Assert((<-aut.Actions).Name, Equals, "groan()")
	c.Assert((<-aut.Actions).Name, Equals, "digest()")
	ch = <-aut.Changes
	c.Assert(ch.Old, Equals, "Eating")
	c.Assert(ch.New, Equals, "Full")
	c.Assert(ch.Duration < time.Millisecond, Equals, true)

	time.Sleep(time.Millisecond)

	// event
	dog.Process("run")
	c.Assert(dog.State.Name, Equals, "Hungry")
	ch = <-aut.Changes
	c.Assert(ch.Old, Equals, "Full")
	c.Assert(ch.New, Equals, "Hungry")
	c.Assert(ch.Duration > time.Millisecond, Equals, true)
}

func (s *S) TestString(c *C) {
	aut, _ := LoadFile("examples/simple.yaml")
	c.Assert(aut.String(), Matches, "simple: Hungry for .*")

	aut.Process("food.meat")
	c.Assert(aut.String(), Matches, "simple: Eating for .*")
}

func (s *S) TestIgnoredEvent(c *C) {
	aut, err := LoadFile("examples/simple.yaml")
	c.Assert(err, Equals, nil)
	dog, ok := aut.Automaton["simple"]
	c.Assert(ok, Equals, true)
	c.Assert(dog.State.Name, Equals, "Hungry")

	// non-event
	dog.Process("blob")
	c.Assert(dog.State.Name, Equals, "Hungry")
}

func (s *S) TestWildcardEvent(c *C) {
	aut, err := LoadFile("examples/simple.yaml")
	c.Assert(err, Equals, nil)
	dog, ok := aut.Automaton["simple"]
	c.Assert(ok, Equals, true)
	c.Assert(dog.State.Name, Equals, "Hungry")

	// event caught by wildcard
	dog.Process("itch.scratch")
	c.Assert(dog.State.Name, Equals, "Hungry")
	c.Assert((<-aut.Actions).String(), Equals, "scratch()")

	// event caught by wildcard
	dog.Process("sniff.nose")
	c.Assert(dog.State.Name, Equals, "Hungry")
	c.Assert((<-aut.Actions).Name, Equals, "sniff()")
}

func (s *S) TestEvent(c *C) {
	aut, err := LoadFile("examples/simple.yaml")
	c.Assert(err, Equals, nil)
	dog, ok := aut.Automaton["simple"]
	c.Assert(ok, Equals, true)
	c.Assert(dog.State.Name, Equals, "Hungry")

	// non-event
	dog.Process("blob")
	c.Assert(dog.State.Name, Equals, "Hungry")
}

// Check reentering the same state does not run leaving/entering actions.
func (s *S) TestReenter(c *C) {
	aut, _ := LoadFile("examples/simple.yaml")

	// event
	aut.Process("food.meat")
	c.Assert((<-aut.Actions).Name, Equals, "woof()")
	c.Assert((<-aut.Actions).Name, Equals, "eat('apple')")
	ch := <-aut.Changes
	c.Assert(ch.Old, Equals, "Hungry")
	c.Assert(ch.New, Equals, "Eating")

	// reenter
	aut.Process("sniff.tree")
	c.Assert((<-aut.Actions).Name, Equals, "sniff()")

	// reenter again
	aut.Process("sniff.tree")
	c.Assert((<-aut.Actions).Name, Equals, "sniff()")

	// migrate
	aut.Process("food.meat")
	c.Assert((<-aut.Actions).Name, Equals, "groan()")
	c.Assert((<-aut.Actions).Name, Equals, "digest()")

	// reenter
	aut.Process("sniff.tree")
	c.Assert((<-aut.Actions).Name, Equals, "sniff()")
}

func (s *S) TestPersistRestore(c *C) {
	aut, err := LoadFile("examples/simple.yaml")
	c.Assert(err, Equals, nil)
	dog, _ := aut.Automaton["simple"]
	c.Assert(dog.State.Name, Equals, "Hungry")

	p := aut.Persist()

	aut, err = LoadFile("examples/simple.yaml")
	aut.Restore(p)
	dog, _ = aut.Automaton["simple"]
	c.Assert(dog.State.Name, Equals, "Hungry")
}

func (s *S) TestRestore(c *C) {
	ps := AutomataState{"simple": AutomatonState{State: "Eating", Since: time.Now()}}

	aut, _ := LoadFile("examples/simple.yaml")
	aut.Restore(ps)
	dog, _ := aut.Automaton["simple"]
	c.Assert(dog.State.Name, Equals, "Eating")
}

func (s *S) TestRestoreInvalid(c *C) {
	// restoring bad state should be ignored
	ps := AutomataState{"simple": AutomatonState{State: "Invalid", Since: time.Now()}}

	aut, _ := LoadFile("examples/simple.yaml")
	aut.Restore(ps)
	dog, _ := aut.Automaton["simple"]
	c.Assert(dog.State.Name, Equals, "Hungry")
}

func (s *S) TestInvalid(c *C) {
	conf := "invalid: {}"
	_, err := Load([]byte(conf))
	c.Assert(err, NotNil)
}
