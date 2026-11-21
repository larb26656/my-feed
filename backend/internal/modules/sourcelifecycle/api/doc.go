// Package api is the public surface of sourcelifecycle for other modules.
//
// Callers that create or deactivate a Source depend on sourceapi.Lifecycle.
// Bootstrap injects this module's service as that implementation; other modules
// must not import the service package.
package api
