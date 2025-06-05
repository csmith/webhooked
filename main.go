package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/csmith/envflag/v2"
	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
	"log"
	"slices"
	"strings"
	"time"
)

var (
	token       = flag.String("token", "", "GitHub access token")
	owner       = flag.String("owner", "", "User/org whose repositories should be scanned ('' for current user)")
	url         = flag.String("url", "", "URL of hook to install")
	contentType = flag.String("content-type", "json", "Media type the hook accepts ('json' or 'form')")
	secret      = flag.String("secret", "", "Secret key to use for hook validation")
	events      = flag.String("events", "*", "Comma-separated list of events to receive ('*' for all)")
	monitor     = flag.Duration("monitor", 0, "if set, webhooked will not exit and will instead monitor for changes at this interval")
)

type Hooker struct {
	ctx    context.Context
	client *github.Client
	hook   *github.Hook
}

func main() {
	envflag.Parse()

	if *token == "" || *url == "" {
		flag.Usage()
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)

	hooker := &Hooker{
		ctx:    context.Background(),
		client: github.NewClient(tc),
		hook: &github.Hook{
			Config: &github.HookConfig{
				URL:         url,
				ContentType: contentType,
				Secret:      secret,
			},
			Events: strings.Split(*events, ","),
		},
	}

	if *monitor < time.Minute {
		if err := hooker.checkHooks(); err != nil {
			panic(err)
		}
	} else {
		log.Print("Monitoring mode...\n")
		for {
			log.Print("Scanning repositories...\n")
			if err := hooker.checkHooks(); err != nil {
				panic(err)
			}
			time.Sleep(*monitor)
		}
	}
}

// checkHooks iterates through all repositories and checks they have the configured hook installed
func (h *Hooker) checkHooks() error {
	repos, err := h.listRepositories()
	if err != nil {
		return fmt.Errorf("unable to list repositories: %v\n", err)
	}

	for i := range repos {
		hook, err := h.findHook(repos[i])
		if err != nil {
			return fmt.Errorf("unable to scan hooks for %s/%s: %v\n", repos[i].Owner, repos[i].Name, err)
		}

		if hook == nil {
			if err := h.installHook(repos[i]); err != nil {
				return fmt.Errorf("unable to install hook for %s/%s: %v\n", repos[i].Owner, repos[i].Name, err)
			} else {
				log.Printf("Installed hook for %s/%s\n", repos[i].Owner, repos[i].Name)
			}
		} else if !h.validHook(hook) {
			if err := h.updateHook(repos[i], *hook.ID); err != nil {
				return fmt.Errorf("unable to update hook for %s/%s: %v\n", repos[i].Owner, repos[i].Name, err)
			} else {
				log.Printf("Updated hook for %s/%s\n", repos[i].Owner, repos[i].Name)
			}
		}
	}
	return nil
}

func (h *Hooker) validHook(hook *github.Hook) bool {
	// Of the config parameters, URL must be the same already and secret is blanked, leaving just content_type.
	if *hook.Config.ContentType != *h.hook.Config.ContentType {
		return false
	}

	if len(hook.Events) != len(h.hook.Events) {
		return false
	}

	// If the lengths are equal, then every one of the installed events should appear in our desired list.
	// We don't care about order, so iterate through them all to check.
	for _, installed := range hook.Events {
		if !slices.Contains(h.hook.Events, installed) {
			return false
		}
	}

	return true
}

type Repo struct {
	Owner string
	Name  string
}

// listRepositories lists all repositories owned by the user
func (h *Hooker) listRepositories() (repos []Repo, err error) {
	err = crawlPages(func(listOptions github.ListOptions) (*github.Response, error) {
		var list []*github.Repository
		var res *github.Response
		var err error

		if *owner == "" {
			list, res, err = h.client.Repositories.ListByAuthenticatedUser(h.ctx, &github.RepositoryListByAuthenticatedUserOptions{
				Type:        "owner",
				ListOptions: listOptions,
			})
		} else {
			list, res, err = h.client.Repositories.ListByUser(h.ctx, *owner, &github.RepositoryListByUserOptions{
				Type:        "owner",
				ListOptions: listOptions,
			})
		}

		if err != nil {
			return nil, err
		}

		for i := range list {
			if list[i].Archived == nil || !*list[i].Archived {
				repos = append(repos, Repo{
					Owner: *list[i].Owner.Login,
					Name:  *list[i].Name,
				})
			}
		}

		return res, nil
	})
	return
}

// listHooks list all installed web hooks for a repository
func (h *Hooker) listHooks(repo Repo) (hooks []*github.Hook, err error) {
	err = crawlPages(func(listOptions github.ListOptions) (*github.Response, error) {
		list, res, err := h.client.Repositories.ListHooks(h.ctx, repo.Owner, repo.Name, &listOptions)

		if err != nil {
			return nil, err
		}

		for i := range list {
			hooks = append(hooks, list[i])
		}

		return res, nil
	})
	return
}

// findHook iterates through the given repository's hooks, and returns the first with a matching URL (or nil)
func (h *Hooker) findHook(repo Repo) (*github.Hook, error) {
	hooks, err := h.listHooks(repo)
	if err != nil {
		return nil, err
	}

	for i := range hooks {
		if *hooks[i].Config.URL == *h.hook.Config.URL {
			return hooks[i], nil
		}
	}

	return nil, nil
}

// installHook installs a new hook configured according to the command line arguments
func (h *Hooker) installHook(repo Repo) error {
	_, _, err := h.client.Repositories.CreateHook(h.ctx, repo.Owner, repo.Name, h.hook)
	return err
}

// updateHook updates the hook with the given ID to be configured according to the command line arguments
func (h *Hooker) updateHook(repo Repo, id int64) error {
	_, _, err := h.client.Repositories.EditHook(h.ctx, repo.Owner, repo.Name, id, h.hook)
	return err
}

// crawlPages repeatedly invokes the given crawler function, paginating through results
func crawlPages(caller func(options github.ListOptions) (*github.Response, error)) error {
	page := 0

	for {
		res, err := caller(github.ListOptions{
			Page:    page,
			PerPage: 50,
		})

		if err != nil {
			return err
		}

		if res.NextPage > 0 {
			page = res.NextPage
		} else {
			return nil
		}
	}
}
