import Lean

set_option warningAsError true

namespace Antifraud.Authorization

inductive Phase where
  | idle
  | needsConfirmation
  | allowed
  | applied
  deriving DecidableEq

/-- One fixed account, operation, and destination. A decision generation
    increases whenever evaluation replaces the current decision. -/
structure State where
  generation : Nat
  phase : Phase
  applications : Nat

inductive Event where
  | evaluateAllow
  | requireStepUp
  | confirm (generation : Nat)
  | deliver (generation : Nat)

def step (state : State) : Event → State
  | .evaluateAllow => { state with generation := state.generation + 1, phase := .allowed }
  | .requireStepUp => { state with generation := state.generation + 1, phase := .needsConfirmation }
  | .confirm generation =>
    if state.phase = .needsConfirmation ∧ generation = state.generation then
      { state with phase := .allowed }
    else state
  | .deliver generation =>
    if state.phase = .allowed ∧ generation = state.generation then
      { state with phase := .applied, applications := state.applications + 1 }
    else state

def run (state : State) (events : List Event) : State :=
  match events with
  | [] => state
  | event :: rest => run (step state event) rest

private theorem stepGenerationMonotone (state : State) (event : Event) :
    state.generation ≤ (step state event).generation := by
  cases event with
  | evaluateAllow => exact Nat.le_succ state.generation
  | requireStepUp => exact Nat.le_succ state.generation
  | confirm generation =>
    simp only [step]
    split <;> exact Nat.le_refl state.generation
  | deliver generation =>
    simp only [step]
    split <;> exact Nat.le_refl state.generation

private theorem runGenerationMonotone (state : State) (events : List Event) :
    state.generation ≤ (run state events).generation := by
  induction events generalizing state with
  | nil => exact Nat.le_refl state.generation
  | cons event rest ih =>
    exact Nat.le_trans (stepGenerationMonotone state event) (ih (step state event))

/-- Once replaced by a step-up decision, a permit from the previous or any
    older decision cannot change state, even after arbitrary later events. -/
theorem supersededPermitCannotApply
    (state : State) (oldGeneration : Nat) (laterEvents : List Event)
    (wasIssuedBefore : oldGeneration ≤ state.generation) :
    let current := run (step state .requireStepUp) laterEvents
    step current (.deliver oldGeneration) = current := by
  dsimp only
  let replaced := step state .requireStepUp
  let current := run replaced laterEvents
  change step current (.deliver oldGeneration) = current
  have olderThanReplacement : oldGeneration < replaced.generation := by
    change oldGeneration < state.generation + 1
    exact Nat.lt_of_le_of_lt wasIssuedBefore (Nat.lt_succ_self state.generation)
  have olderThanCurrent : oldGeneration < current.generation :=
    Nat.lt_of_lt_of_le olderThanReplacement (runGenerationMonotone replaced laterEvents)
  have invalid : ¬ (current.phase = .allowed ∧ oldGeneration = current.generation) :=
    fun authorization => (Nat.ne_of_lt olderThanCurrent) authorization.2
  rw [step, ite_eq_right invalid]

/-- A valid confirmation and its delivery can apply the operation once in
    the model when these two events run without an intervening replacement. -/
theorem currentConfirmationCanApply (state : State)
    (pending : state.phase = .needsConfirmation) :
    let final := run state [.confirm state.generation, .deliver state.generation]
    final.phase = .applied ∧ final.applications = state.applications + 1 := by
  simp [run, step, pending]

example :
    (run ⟨0, .idle, 0⟩ [.requireStepUp, .confirm 1, .deliver 1]).applications = 1 := by
  decide

/-- info: 'Antifraud.Authorization.supersededPermitCannotApply' does not depend on any axioms -/
#guard_msgs in
#print axioms supersededPermitCannotApply

/-- info: 'Antifraud.Authorization.currentConfirmationCanApply' depends on axioms: [propext] -/
#guard_msgs in
#print axioms currentConfirmationCanApply

end Antifraud.Authorization
