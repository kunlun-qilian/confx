package confx

import (
	"bytes"
	"fmt"
	"path/filepath"
)

type DockerConfig struct {
	BuildImage   string
	RuntimeImage string
	GoProxy      GoProxyConfig
	Openapi      bool
}

type GoProxyConfig struct {
	ProxyOn bool
	Host    string
}

func (c *DockerConfig) setDefaults() {
	if c.BuildImage == "" {
		c.BuildImage = "golang:1.20-buster"
	}
	if c.RuntimeImage == "" {
		c.RuntimeImage = "alpine"
	}
	if c.GoProxy.ProxyOn {
		if c.GoProxy.Host == "" {
			c.GoProxy.Host = "https://goproxy.cn,direct"
		}
	}
}

func (c *Configuration) dockerfile() []byte {
	c.dockerConfig.setDefaults()
	dockerfile := bytes.NewBuffer(nil)
	// builder
	_, _ = fmt.Fprintf(dockerfile, "FROM %s AS build-env\n", c.dockerConfig.BuildImage)

	_, _ = fmt.Fprintln(dockerfile, `
FROM build-env AS builder
`)
	// go proxy
	if c.dockerConfig.GoProxy.ProxyOn {
		_, _ = fmt.Fprintln(dockerfile, fmt.Sprintf(`
ARG GOPROXY=%s`, c.dockerConfig.GoProxy.Host))
	}

	_, _ = fmt.Fprintln(dockerfile, `
WORKDIR /go/src
COPY ./ ./

# build
RUN make build WORKSPACE=`+c.WorkSpace())

	// runtime
	_, _ = fmt.Fprintln(dockerfile, fmt.Sprintf(
		`
# runtime
FROM %s`, c.dockerConfig.RuntimeImage))
	_, _ = fmt.Fprintln(dockerfile, `
COPY --from=builder `+ShouldReplacePath(filepath.Join("/go/src/cmd", c.WorkSpace(), c.WorkSpace()))+` `+ShouldReplacePath(filepath.Join(`/go/bin`, c.Command.Use))+`
`)
	if c.dockerConfig.Openapi {
		// openapi 3.0
		_, _ = fmt.Fprintln(dockerfile,
			`
# openapi 3.0
COPY --from=builder `+
				ShouldReplacePath(filepath.Join("/go/src/cmd", c.WorkSpace(), "openapi.json"))+` `+ShouldReplacePath(filepath.Join("/go/bin", "openapi.json")))
		// gin swagger
		_, _ = fmt.Fprintln(dockerfile,
			`
# gin swagger 2.0
COPY --from=builder `+
				ShouldReplacePath(filepath.Join("/go/src/cmd", c.WorkSpace(), "docs"))+` `+ShouldReplacePath(filepath.Join("/go/bin", "docs")))
	}

	for _, envVar := range c.defaultEnvVars.Values {
		if envVar.Value != "" {
			if envVar.IsExpose {
				_, _ = fmt.Fprintln(dockerfile, `
EXPOSE`, envVar.Value)
			}
		}
	}

	fmt.Fprintf(dockerfile, `
ARG PROJECT_NAME
ARG PROJECT_VERSION
ENV PROJECT_NAME=${PROJECT_NAME} PROJECT_VERSION=${PROJECT_VERSION}

WORKDIR /go/bin
ENTRYPOINT ["`+ShouldReplacePath(filepath.Join(`/go/bin`, c.Command.Use))+`"]
`)

	return dockerfile.Bytes()
}
