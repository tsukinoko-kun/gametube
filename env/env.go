package env

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func EnsureCommonEnv() error {
	if _, ok := os.LookupEnv("XDG_CONFIG_HOME"); !ok {
		if configDir, err := os.UserConfigDir(); err != nil {
			return err
		} else {
			if err := os.Setenv("XDG_CONFIG_HOME", configDir); err != nil {
				return err
			}
		}
	}

	if _, ok := os.LookupEnv("XDG_CACHE_HOME"); !ok {
		if cacheDir, err := os.UserCacheDir(); err != nil {
			return err
		} else {
			if err := os.Setenv("XDG_CACHE_HOME", cacheDir); err != nil {
				return err
			}
		}
	}

	if _, ok := os.LookupEnv("XDG_CONFIG_HOME"); !ok {
		if home, err := os.UserHomeDir(); err != nil {
			return err
		} else {
			if err := os.Setenv("XDG_DATA_HOME", filepath.Join(home, ".config")); err != nil {
				return err
			}
		}
	}

	if _, ok := os.LookupEnv("XDG_DATA_HOME"); !ok {
		if home, err := os.UserHomeDir(); err != nil {
			return err
		} else {
			if err := os.Setenv("XDG_DATA_HOME", filepath.Join(home, ".config")); err != nil {
				return err
			}
		}
	}

	if _, ok := os.LookupEnv("XDG_STATE_HOME"); !ok {
		if home, err := os.UserHomeDir(); err != nil {
			return err
		} else {
			if err := os.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state")); err != nil {
				return err
			}
		}
	}

	if _, ok := os.LookupEnv("XDG_RUNTIME_DIR"); !ok {
		if home, err := os.UserHomeDir(); err != nil {
			return err
		} else {
			if err := os.Setenv("XDG_RUNTIME_DIR", filepath.Join(home, ".local", "run")); err != nil {
				return err
			}
		}
	}

	return nil
}

func dynamicMapping(s string) string {
	switch s {
	case "NOW":
		return fmt.Sprintf("%d", time.Now().Unix())
	}
	return ""
}

func Expand(in string) string {
	return os.Expand(os.ExpandEnv(in), dynamicMapping)
}
