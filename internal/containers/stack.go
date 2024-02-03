package containers

type (
	Stack[T any] struct {
		top    *node[T]
		length int
	}
	node[T any] struct {
		value T
		prev  *node[T]
	}
)

// Create a new stack
func New[T any]() *Stack[T] {
	return &Stack[T]{nil, 0}
}

// Return the number of items in the stack
func (stack *Stack[T]) Len() int {
	return stack.length
}

// View the top item on the stack
func (stack *Stack[T]) Peek() (result T) {
	if stack.length == 0 {
		return
	}
	return stack.top.value
}

// Pop the top item of the stack and return it
func (stack *Stack[T]) Pop() (result T) {
	if stack.length == 0 {
		return
	}

	n := stack.top
	stack.top = n.prev
	stack.length--
	return n.value
}

// Push a value onto the top of the stack
func (stack *Stack[T]) Push(value T) {
	n := &node[T]{value, stack.top}
	stack.top = n
	stack.length++
}
