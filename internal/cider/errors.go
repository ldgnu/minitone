// minitone - TUI pa' controlar Apple Music desde Cider
// by ldgnu <ldgnu@users.noreply.github.com>
// Usalo, rompelo, mejoralo — total, pa' eso estamos

package cider

import "errors"

var (
	ErrNotRunning   = errors.New("Cider is not running or RPC is disabled")
	ErrUnauthorized = errors.New("invalid or missing API token")
	ErrNotFound     = errors.New("resource not found")
)
