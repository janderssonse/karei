// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package system

// File and directory permission constants for consistent usage across the codebase.
const (
	// FilePermUserRW allows read/write for user only (0600).
	FilePermUserRW = 0600

	// FilePermUserRWGroupR allows read/write for user, read for group (0640).
	FilePermUserRWGroupR = 0640

	// FilePermDefault is the default file permission (0644).
	FilePermDefault = 0644

	// FilePermExecutable is for executable files (0755).
	FilePermExecutable = 0755

	// DirPermUserOnly allows full access for user only (0700).
	DirPermUserOnly = 0700

	// DirPermDefault is the default directory permission (0755).
	DirPermDefault = 0755
)
