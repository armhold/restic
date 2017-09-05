package web

// choose which paths (files/directories) to include/exclude from backup

type BackupPaths struct {
	BackupPaths []string
	Excludes    []string
}

func (b *BackupPaths) Validate() (ok bool, errors FormErrors) {
	errors = make(map[string]string)

	return len(errors) == 0, errors
}
