package web

import (
	"github.com/restic/restic/internal/restic"
	"context"
	"github.com/restic/restic/internal/options"
	"github.com/restic/restic/internal/backend/s3"
	"github.com/restic/restic/internal/backend/gs"
	"github.com/restic/restic/internal/backend/rest"
	"github.com/restic/restic/internal/debug"
	"github.com/restic/restic/internal/backend/location"
	"github.com/restic/restic/internal/backend/local"
	"github.com/restic/restic/internal/backend/azure"
	"github.com/restic/restic/internal/backend/swift"
	"github.com/restic/restic/internal/backend/b2"
	"io/ioutil"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/backend/sftp"
	"os"
	"github.com/restic/restic/internal/repository"
)

// TODO: rename this file
// TODO: much copied from global.go, changed slightly for use as a web server with a stored configuration


func parseConfig(loc location.Location, opts options.Options) (interface{}, error) {
	// only apply options for a particular backend here
	opts = opts.Extract(loc.Scheme)

	switch loc.Scheme {
	case "local":
		cfg := loc.Config.(local.Config)
		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening local repository at %#v", cfg)
		return cfg, nil

	case "sftp":
		cfg := loc.Config.(sftp.Config)
		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening sftp repository at %#v", cfg)
		return cfg, nil

	case "s3":
		cfg := loc.Config.(s3.Config)
		if cfg.KeyID == "" {
			cfg.KeyID = os.Getenv("AWS_ACCESS_KEY_ID")
		}

		if cfg.Secret == "" {
			cfg.Secret = os.Getenv("AWS_SECRET_ACCESS_KEY")
		}

		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening s3 repository at %#v", cfg)
		return cfg, nil

	case "gs":
		cfg := loc.Config.(gs.Config)
		if cfg.ProjectID == "" {
			cfg.ProjectID = os.Getenv("GOOGLE_PROJECT_ID")
		}

		if cfg.JSONKeyPath == "" {
			if path := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); path != "" {
				// Check read access
				if _, err := ioutil.ReadFile(path); err != nil {
					return nil, errors.Fatalf("Failed to read google credential from file %v: %v", path, err)
				}
				cfg.JSONKeyPath = path
			} else {
				return nil, errors.Fatal("No credential file path is set")
			}
		}

		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening gs repository at %#v", cfg)
		return cfg, nil

	case "azure":
		cfg := loc.Config.(azure.Config)
		if cfg.AccountName == "" {
			cfg.AccountName = os.Getenv("AZURE_ACCOUNT_NAME")
		}

		if cfg.AccountKey == "" {
			cfg.AccountKey = os.Getenv("AZURE_ACCOUNT_KEY")
		}

		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening gs repository at %#v", cfg)
		return cfg, nil

	case "swift":
		cfg := loc.Config.(swift.Config)

		if err := swift.ApplyEnvironment("", &cfg); err != nil {
			return nil, err
		}

		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening swift repository at %#v", cfg)
		return cfg, nil

	case "b2":
		cfg := loc.Config.(b2.Config)

		if cfg.AccountID == "" {
			cfg.AccountID = os.Getenv("B2_ACCOUNT_ID")
		}

		if cfg.Key == "" {
			cfg.Key = os.Getenv("B2_ACCOUNT_KEY")
		}

		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening b2 repository at %#v", cfg)
		return cfg, nil
	case "rest":
		cfg := loc.Config.(rest.Config)
		if err := opts.Apply(loc.Scheme, &cfg); err != nil {
			return nil, err
		}

		debug.Log("opening rest repository at %#v", cfg)
		return cfg, nil
	}

	return nil, errors.Fatalf("invalid backend: %q", loc.Scheme)
}

// Open the backend specified by a location config.
func open(s string, opts options.Options) (restic.Backend, error) {
	debug.Log("parsing location %v", s)
	loc, err := location.Parse(s)
	if err != nil {
		return nil, errors.Fatalf("parsing repository location failed: %v", err)
	}

	var be restic.Backend

	cfg, err := parseConfig(loc, opts)
	if err != nil {
		return nil, err
	}

	switch loc.Scheme {
	case "local":
		be, err = local.Open(cfg.(local.Config))
	case "sftp":
		be, err = sftp.Open(cfg.(sftp.Config))
	case "s3":
		be, err = s3.Open(cfg.(s3.Config))
	case "gs":
		be, err = gs.Open(cfg.(gs.Config))
	case "azure":
		be, err = azure.Open(cfg.(azure.Config))
	case "swift":
		be, err = swift.Open(cfg.(swift.Config))
	case "b2":
		be, err = b2.Open(cfg.(b2.Config))
	case "rest":
		be, err = rest.Open(cfg.(rest.Config))

	default:
		return nil, errors.Fatalf("invalid backend: %q", loc.Scheme)
	}

	if err != nil {
		return nil, errors.Fatalf("unable to open repo at %v: %v", s, err)
	}

	// check if config is there
	fi, err := be.Stat(context.TODO(), restic.Handle{Type: restic.ConfigFile})
	if err != nil {
		return nil, errors.Fatalf("unable to open config file: %v\nIs there a repository at the following location?\n%v", err, s)
	}

	if fi.Size == 0 {
		return nil, errors.New("config file has zero size, invalid repository?")
	}

	return be, nil
}

const maxKeys = 20

// OpenRepository reads the password and opens the repository.
func OpenRepository(path, password string) (*repository.Repository, error) {
	opts := make(options.Options)
	//opts

	be, err := open(path, opts)
	if err != nil {
		return nil, err
	}

	s := repository.New(be)

	err = s.SearchKey(context.TODO(), password, maxKeys)
	if err != nil {
		return nil, errors.Fatalf("unable to open repo: %v", err)
	}

	return s, nil
}

