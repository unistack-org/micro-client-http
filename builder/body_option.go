package builder

const (
	singleWildcard string = "*"
	doubleWildcard string = "**"
)

type bodyOption string

func (o bodyOption) String() string { return string(o) }

func (o bodyOption) isFullBody() bool {
	return o.String() == singleWildcard
}

func (o bodyOption) isWithoutBody() bool {
	return o == ""
}

func (o bodyOption) isSingleField() bool {
	return o != "" && o.String() != singleWildcard
}
