module status/tests

go 1.21

replace status => ../../backend

require (
	example.com/stretchr/testify v1.8.4
	status v0.0.0-00010101000000-000000000000
)

require (
	example.com/davecgh/go-spew v1.1.1 // indirect
	example.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
