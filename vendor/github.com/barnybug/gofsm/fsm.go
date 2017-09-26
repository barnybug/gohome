// Package gofsm is a library for building finite-state machines (automata).
//
// Why yet another FSM library for go?
//
// gofsm is aimed squarely at home automation - human visible, configured and
// friendly. The configuration format is yaml, and easy to read/write by hand.
//
// gofsm is used in the gohome automation project:
// http://github.com/barnybug/gohome
package gofsm

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"gopkg.in/yaml.v1"
)

type Automata struct {
	Automaton map[string]*Automaton
	Actions   chan Action
	Changes   chan Change
}

type State struct {
	Name     string
	Steps    []Step
	Entering Actions
	Leaving  Actions
}

type Step struct {
	When    string
	Actions Actions
	Next    string
}

type Actions []string

type Transition struct {
	When    string
	Actions Actions
}

type Automaton struct {
	Start  string
	States map[string]struct {
		Entering Actions
		Leaving  Actions
	}
	Transitions map[string][]Transition
	Name        string
	State       *State
	Since       time.Time
	actions     chan Action
	changes     chan Change
	sm          map[string]*State
}

type Action struct {
	Name    string
	Trigger interface{}
	Change  Change
}

type Change struct {
	Automaton string
	Old       string
	New       string
	Since     time.Time
	Duration  time.Duration
	Trigger   interface{}
}

type Event interface {
	Match(s string) bool
}

func (self Action) String() string {
	return self.Name
}

func (self *Automata) Process(event Event) {
	for _, aut := range self.Automaton {
		aut.Process(event)
	}
}

func (self *Automata) String() string {
	var out string
	now := time.Now()
	for k, aut := range self.Automaton {
		if out != "" {
			out += "\n"
		}
		du := now.Sub(aut.Since)
		out += fmt.Sprintf("%s: %s for %s", k, aut.State.Name, du)
	}
	return out
}

func (self *Automaton) Process(event Event) {
	for _, t := range self.State.Steps {
		if event.Match(t.When) {
			now := time.Now()
			var change Change
			// is a state change happening
			stateChanged := (self.State.Name != t.Next)

			if stateChanged {
				duration := now.Sub(self.Since)
				change = Change{Automaton: self.Name, Old: self.State.Name, New: t.Next, Duration: duration, Since: self.Since, Trigger: event}

				// emit leaving actions
				for _, action := range self.State.Leaving {
					self.actions <- Action{action, event, change}
				}
			}

			// emit transition actions
			for _, action := range t.Actions {
				self.actions <- Action{action, event, change}
			}

			// change state
			if stateChanged {
				self.State = self.sm[t.Next]
				self.Since = now
				self.changes <- change

				// emit entering actions
				for _, action := range self.State.Entering {
					self.actions <- Action{action, event, change}
				}
			}
		}
	}
}

func (self *Automaton) load() error {
	if self.Start == "" {
		return errors.New("missing Start entry")
	}
	if len(self.States) == 0 {
		return errors.New("missing States entries")
	}
	if len(self.Transitions) == 0 {
		return errors.New("missing Transitions entries")
	}

	sm := map[string]*State{}

	var allStates []string
	for name, val := range self.States {
		state := State{Name: name}
		state.Entering = val.Entering
		state.Leaving = val.Leaving
		sm[name] = &state

		allStates = append(allStates, name)
	}
	self.sm = sm

	var ok bool
	if self.State, ok = sm[self.Start]; !ok {
		return errors.New("starting State invalid")
	}
	self.Since = time.Now()

	type StringPair struct {
		_1 string
		_2 string
	}

	for name, trans := range self.Transitions {
		var pairs []StringPair
		lr := strings.SplitN(name, "->", 2)
		if len(lr) == 2 {
			// from->to
			var froms, tos []string
			if lr[0] == "*" {
				froms = allStates
			} else {
				froms = strings.Split(lr[0], ",")
			}
			if lr[1] == "*" {
				tos = allStates
			} else {
				tos = strings.Split(lr[1], ",")
			}
			for _, f := range froms {
				for _, t := range tos {
					pairs = append(pairs, StringPair{f, t})
				}
			}
		} else {
			// from1,from2 = from1->from1, from2->from2
			var froms []string
			if lr[0] == "*" {
				froms = allStates
			} else {
				froms = strings.Split(lr[0], ",")
			}
			for _, f := range froms {
				pairs = append(pairs, StringPair{f, f})
			}
		}

		for _, pair := range pairs {
			from, to := pair._1, pair._2
			var sfrom *State
			if sfrom, ok = self.sm[from]; !ok {
				return errors.New(fmt.Sprintf("State: %s not found", from))
			}
			if _, ok := self.sm[to]; !ok {
				return errors.New(fmt.Sprintf("State: %s not found", from))
			}

			for _, v := range trans {
				t := Step{v.When, v.Actions, to}
				sfrom.Steps = append(sfrom.Steps, t)
			}
		}
	}
	return nil
}

type AutomataState map[string]AutomatonState

type AutomatonState struct {
	State string
	Since time.Time
}

func (self *Automata) Persist() AutomataState {
	ret := AutomataState{}
	for k, aut := range self.Automaton {
		ret[k] = AutomatonState{aut.State.Name, aut.Since}
	}
	return ret
}

func (self *Automata) Restore(s AutomataState) {
	for k, as := range s {
		if aut, ok := self.Automaton[k]; ok {
			if state, ok := aut.sm[as.State]; ok {
				aut.State = state
				aut.Since = as.Since
			} else {
				log.Println("Invalid restored state:", as.State)
			}
		}
	}
}

func LoadFile(filename string) (*Automata, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Load(data)
}

func Load(str []byte) (*Automata, error) {
	var aut Automata = Automata{Actions: make(chan Action, 32), Changes: make(chan Change, 32)}
	err := yaml.Unmarshal(str, &aut.Automaton)
	for k, a := range aut.Automaton {
		err := a.load()
		if err != nil {
			return nil, errors.New(fmt.Sprintf("%s: %s", k, err.Error()))
		}
		a.Name = k
		a.actions = aut.Actions
		a.changes = aut.Changes
	}

	return &aut, err
}
