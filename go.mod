module github.com/Melkor333/oils-readline

go 1.23.6

toolchain go1.24.2

//replace mvdan.cc/sh/v3 => ../mvdan-sh

require (
	github.com/creack/pty v1.1.24
	github.com/mcpherrinm/multireader v0.0.0-20210209030331-ecd0fad39ad6
	github.com/muesli/cancelreader v0.2.2
	github.com/reeflective/readline v1.1.3
	golang.org/x/sys v0.33.0
)

require (
	github.com/rivo/uniseg v0.4.7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
