package proto

// FragmentOptions configures the fragmentation
type FragmentOption struct {
	MTU uint
}

func NewFragmentOption() *FragmentOption {
	return &FragmentOption{0}
}

func (fragment *FragmentOption) Enabled() bool {
	return fragment.MTU > 0
}
