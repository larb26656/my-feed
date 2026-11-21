# API documentation

The OpenAPI document is generated from HTTP handler annotations and Go DTOs. Run `make openapi` after changing the API and commit the updated document. `make test` requires Python 3 and Node.js, validates the OpenAPI document, and checks that it is current; the router tests check coverage of `/api/v1` and `/healthz`.

The bundled `assets/scalar.js` comes from `@scalar/api-reference` version 1.82.1 (MIT, https://github.com/scalar/scalar/blob/main/LICENSE). It is served locally so the API reference does not need a CDN.
