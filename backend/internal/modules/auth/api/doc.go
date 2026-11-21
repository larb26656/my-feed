// Package api is the public surface of auth for other modules.
//
// Auth is consumed through HTTP and bootstrap wiring today. Keep this package
// so the module has an explicit public boundary if other modules need Session
// operations later.
package api
