package kind

type ResourceYaml interface {
	RenderYaml() (bytes []byte, err error)
}
