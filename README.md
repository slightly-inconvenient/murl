# MURL

MURL is a simple redirection service.

It's primary motivation was to simplify organization internal urls by e.g. codifying simple aliases for company internal service urls and make legacy urls work when switching to newer versions or different providers for organization internal services. It is however well suited for most use cases where a simple parameterized redirect to another URL is sufficient.

MURL requires all URL mappings to be pre-registered through configuration. 

Each url mapping supports:
- A path to match against with variable extraction using any supported [go http.ServeMux pattern](https://pkg.go.dev/net/http#hdr-Patterns-ServeMux)
- Extracting params from request path, query params or headers and from a per-route allowlisted subset of environment using [templates](https://pkg.go.dev/text/template)
- Checking extracted params using the [Common Expression Language](https://github.com/google/cel-go)
- Building a redirect URL using [templates](https://pkg.go.dev/text/template)

## Configuration

See [cmd/testdata/config.yaml](./cmd/testdata/config.yaml) for all supported configuration values with explanations.

JSON based configuration files are also supported in addition to YAML.

## Usage

MURL is available either as a binary through [GitHub Releases](https://github.com/slightly-inconvenient/murl/releases/latest) or as a multiarch OCI image through GitHub Container Registry [packages](https://github.com/slightly-inconvenient/murl/pkgs/container/murl).

### Binary

To serve routes run the binary with 
```sh
murl serve --config /path/to/config.yaml
```

To see all supported commands run the binary with
```sh
murl --help
```

### Container 

Pull the image with
```sh
docker pull ghcr.io/slightly-inconvenient/murl:latest
```

Run the image 
```sh
docker run --read-only -v /path/to/config.yaml:/app/config.yaml:ro ghcr.io/slightly-inconvenient/murl:latest serve --config /app/config.yaml
```