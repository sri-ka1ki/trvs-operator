package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"io/ioutil"
	"os"
	"path"
	"time"
)

var keychainsPath = os.Getenv("TRAVIS_KEYCHAIN_DIR")

func NewKeychain(name, repoURL string, key []byte) (*Keychain, error) {
	keys, err := ssh.NewPublicKeys("git", key, "")
	if err != nil {
		return nil, err
	}

	k := &Keychain{
		Name:          name,
		RepositoryURL: repoURL,
		Keys:          keys,
	}

	if err = k.initialize(); err != nil {
		return nil, err
	}

	return k, nil
}

type Keychain struct {
	Name          string
	Path          string
	RepositoryURL string
	Keys          *ssh.PublicKeys
	Repository    *git.Repository
}

func (k *Keychain) initialize() error {
	if keychainsPath == "" {
		return fmt.Errorf("keychains path is empty")
	}

	if err := os.MkdirAll(keychainsPath, 0777); err != nil {
		return err
	}

	k.Path = path.Join(keychainsPath, k.Name)
	r, err := git.PlainOpen(k.Path)
	if err != nil {
		if err == git.ErrRepositoryNotExists {
			r, err = k.clone()
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	k.Repository = r
	if err = k.Update(); err != nil {
		return err
	}

	return nil
}

func (k *Keychain) clone() (*git.Repository, error) {
	if k.RepositoryURL == "" {
		return nil, fmt.Errorf("a templates URL is required when templates are not already cloned")
	}

	entry := log.WithFields(log.Fields{
		"path": k.Path,
		"url":  k.RepositoryURL,
	})

	r, err := git.PlainClone(k.Path, false, &git.CloneOptions{
		URL:  k.RepositoryURL,
		Auth: k.Keys,
	})
	if err != nil {
		entry.WithError(err).Error("could not clone keychain")
		return nil, err
	}

	entry.Info("cloned keychain")
	return r, nil
}

func (k *Keychain) Update() error {
	entry := log.WithFields(log.Fields{
		"path": k.Path,
		"url":  k.RepositoryURL,
	})

	wt, err := k.Repository.Worktree()
	if err != nil {
		return err
	}

	if err := wt.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       k.Keys,
		Force:      true,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			entry.WithError(err).Error("could not update keychain")
			return err
		}
	} else {
		entry.Info("updated keychain")
	}

	return nil
}

func (k *Keychain) Watch(d time.Duration) {
	for {
		k.Update()
		time.Sleep(d)
	}
}

func (k *Keychain) ReadFile(file string) ([]byte, error) {
	fullPath := path.Join(k.Path, file)
	return ioutil.ReadFile(fullPath)
}
