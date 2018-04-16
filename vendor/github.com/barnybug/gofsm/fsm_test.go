package gofsm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type StringEvent string

func (s StringEvent) Match(when string) bool {
	return string(s) == when
}

func TestEvents(t *testing.T) {
	assert := assert.New(t)
	aut, err := LoadFile("examples/simple.yaml")
	assert.NoError(err)
	dog, ok := aut.Automaton["simple"]
	assert.True(ok)
	assert.Equal(dog.State.Name, "Hungry")

	// event
	dog.Process(StringEvent("food"))
	assert.Equal(dog.State.Name, "Eating")
	assert.Equal((<-aut.Actions).Name, "woof()")
	assert.Equal((<-aut.Actions).Name, "eat('apple')")
	ch := <-aut.Changes
	assert.Equal(ch.Old, "Hungry")
	assert.Equal(ch.New, "Eating")
	assert.Equal(ch.Duration < time.Millisecond, true)

	// event
	dog.Process(StringEvent("food"))
	assert.Equal(dog.State.Name, "Full")
	assert.Equal((<-aut.Actions).Name, "groan()")
	assert.Equal((<-aut.Actions).Name, "digest()")
	ch = <-aut.Changes
	assert.Equal(ch.Old, "Eating")
	assert.Equal(ch.New, "Full")
	assert.Equal(ch.Duration < time.Millisecond, true)

	time.Sleep(time.Millisecond)

	// event
	dog.Process(StringEvent("run"))
	assert.Equal(dog.State.Name, "Hungry")
	ch = <-aut.Changes
	assert.Equal(ch.Old, "Full")
	assert.Equal(ch.New, "Hungry")
	assert.Equal(ch.Duration > time.Millisecond, true)
}

func TestChangeState(t *testing.T) {
	assert := assert.New(t)

	aut, err := LoadFile("examples/simple.yaml")
	assert.NoError(err)
	dog := aut.Automaton["simple"]

	dog.ChangeState("Eating", StringEvent("dummy"))
	assert.Equal(dog.State.Name, "Eating")

	dog.ChangeState("Full", StringEvent("dummy"))
	assert.Equal(dog.State.Name, "Full")
	assert.Equal((<-aut.Actions).Name, "groan()")
}

func TestString(t *testing.T) {
	assert := assert.New(t)

	aut, _ := LoadFile("examples/simple.yaml")
	assert.Regexp("simple: Hungry for .*", aut.String())

	aut.Process(StringEvent("food"))
	assert.Regexp("simple: Eating for .*", aut.String())
}

func TestIgnoredEvent(t *testing.T) {
	assert := assert.New(t)

	aut, err := LoadFile("examples/simple.yaml")
	assert.NoError(err)
	dog, ok := aut.Automaton["simple"]
	assert.True(ok)
	assert.Equal(dog.State.Name, "Hungry")

	// non-event
	dog.Process(StringEvent("blob"))
	assert.Equal(dog.State.Name, "Hungry")
}

func TestWildcardEvent(t *testing.T) {
	assert := assert.New(t)

	aut, err := LoadFile("examples/simple.yaml")
	assert.NoError(err)
	dog, ok := aut.Automaton["simple"]
	assert.True(ok)
	assert.Equal(dog.State.Name, "Hungry")

	// event caught by wildcard
	dog.Process(StringEvent("scratch"))
	assert.Equal(dog.State.Name, "Hungry")
	assert.Equal((<-aut.Actions).String(), "scratch()")

	// event caught by wildcard
	dog.Process(StringEvent("sniff"))
	assert.Equal(dog.State.Name, "Hungry")
	assert.Equal((<-aut.Actions).Name, "sniff()")
}

func TestEvent(t *testing.T) {
	assert := assert.New(t)

	aut, err := LoadFile("examples/simple.yaml")
	assert.NoError(err)
	dog, ok := aut.Automaton["simple"]
	assert.True(ok)
	assert.Equal(dog.State.Name, "Hungry")

	// non-event
	dog.Process(StringEvent("blob"))
	assert.Equal(dog.State.Name, "Hungry")
}

// Check reentering the same state does not run leaving/entering actions.
func TestReenter(t *testing.T) {
	assert := assert.New(t)

	aut, _ := LoadFile("examples/simple.yaml")

	// event
	aut.Process(StringEvent("food"))
	assert.Equal((<-aut.Actions).Name, "woof()")
	assert.Equal((<-aut.Actions).Name, "eat('apple')")
	ch := <-aut.Changes
	assert.Equal(ch.Old, "Hungry")
	assert.Equal(ch.New, "Eating")

	// reenter
	aut.Process(StringEvent("sniff"))
	assert.Equal((<-aut.Actions).Name, "sniff()")

	// reenter again
	aut.Process(StringEvent("sniff"))
	assert.Equal((<-aut.Actions).Name, "sniff()")

	// migrate
	aut.Process(StringEvent("food"))
	assert.Equal((<-aut.Actions).Name, "groan()")
	assert.Equal((<-aut.Actions).Name, "digest()")

	// reenter
	aut.Process(StringEvent("sniff"))
	assert.Equal((<-aut.Actions).Name, "sniff()")
}

func TestPersistRestore(t *testing.T) {
	assert := assert.New(t)

	aut, err := LoadFile("examples/simple.yaml")
	assert.NoError(err)
	dog, _ := aut.Automaton["simple"]
	assert.Equal(dog.State.Name, "Hungry")

	p := aut.Persist()

	aut, err = LoadFile("examples/simple.yaml")
	aut.Restore(p)
	dog, _ = aut.Automaton["simple"]
	assert.Equal(dog.State.Name, "Hungry")
}

func TestRestore(t *testing.T) {
	assert := assert.New(t)

	ps := AutomataState{"simple": AutomatonState{State: "Eating", Since: time.Now()}}

	aut, _ := LoadFile("examples/simple.yaml")
	aut.Restore(ps)
	dog, _ := aut.Automaton["simple"]
	assert.Equal(dog.State.Name, "Eating")
}

func TestRestoreInvalid(t *testing.T) {
	assert := assert.New(t)

	// restoring bad state should be ignored
	ps := AutomataState{"simple": AutomatonState{State: "Invalid", Since: time.Now()}}

	aut, _ := LoadFile("examples/simple.yaml")
	aut.Restore(ps)
	dog, _ := aut.Automaton["simple"]
	assert.Equal(dog.State.Name, "Hungry")
}

func TestInvalid(t *testing.T) {
	assert := assert.New(t)

	conf := "invalid: {}"
	_, err := Load([]byte(conf))
	assert.Error(err)
}

func TestAmbiguous(t *testing.T) {
	assert := assert.New(t)

	_, err := LoadFile("examples/ambiguous.yaml")
	assert.Error(err)
}
