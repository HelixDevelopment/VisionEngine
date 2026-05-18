module digital.vasic.visionengine

go 1.25.3

require github.com/stretchr/testify v1.11.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	gocv.io/x/gocv v0.43.0
	golang.org/x/crypto v0.51.0
)

replace digital.vasic.llmprovider => ../LLMProvider
