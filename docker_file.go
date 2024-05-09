package confx

import (
	"bytes"
	"fmt"
	"path/filepath"
)

type DockerConfig struct {
	BuildImage     string
	RuntimeImage   string
	GoConfig       GoConfig
	Openapi        bool
	GitlabCIConfig GitlabCIConfig
}

type GitlabCIConfig struct {
	GitlabCI   bool
	GitlabHost string
}

type GoConfig struct {
	ProxyOn     bool
	ProxyHost   string
	PrivateHost string
}

func (c *DockerConfig) setDefaults() {
	if c.BuildImage == "" {
		c.BuildImage = "golang:1.20-buster"
	}
	if c.RuntimeImage == "" {
		c.RuntimeImage = "alpine"
	}
}

func (c *Configuration) dockerfile() []byte {
	c.dockerConfig.setDefaults()
	dockerfile := bytes.NewBuffer(nil)
	// builder
	_, _ = fmt.Fprintf(dockerfile, "FROM %s AS build-env\n", c.dockerConfig.BuildImage)
	// go proxy
	if c.dockerConfig.GoConfig.ProxyOn {
		if c.dockerConfig.GoConfig.ProxyHost == "" {
			_, _ = fmt.Fprintln(dockerfile, `
ARG GOPROXY`)
		} else {
			_, _ = fmt.Fprintln(dockerfile, fmt.Sprintf(`
ARG GOPROXY=%s`, c.dockerConfig.GoConfig.ProxyHost))
		}
		if c.dockerConfig.GoConfig.PrivateHost == "" {
			_, _ = fmt.Fprintln(dockerfile, `
ARG GOPRIVATE`)
		} else {
			_, _ = fmt.Fprintln(dockerfile, fmt.Sprintf(`
ARG GOPRIVATE=%s`, c.dockerConfig.GoConfig.PrivateHost))
		}
	}
	// gitlab
	if c.dockerConfig.GitlabCIConfig.GitlabCI {
		_, _ = fmt.Fprintln(dockerfile, `
ARG GITLAB_CI_TOKEN`)
		if c.dockerConfig.GitlabCIConfig.GitlabHost == "" {
			panic("GitlabHost is nil")
		}
		_, _ = fmt.Fprintln(dockerfile, fmt.Sprintf(`
ARG GITLAB_HOST=%s`, c.dockerConfig.GitlabCIConfig.GitlabHost))
		_, _ = fmt.Fprintln(dockerfile, `
ENV GONOSUMDB=${GITLAB_HOST}/*`)
		_, _ = fmt.Fprintln(dockerfile, `
RUN git config --global url.https://gitlab-ci-token:${GITLAB_CI_TOKEN}@${GITLAB_HOST}/.insteadOf https://${GITLAB_HOST}/`)
	}
	_, _ = fmt.Fprintln(dockerfile, `
FROM build-env AS builder
`)

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
