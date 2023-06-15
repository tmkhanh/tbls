package schema

import (
	"fmt"
	"strings"

	"github.com/minio/pkg/wildcard"
	"github.com/pkg/errors"
)

type FilterOption struct {
	Include       []string
	Exclude       []string
	IncludeLabels []string
	Distance      int
}

func (s *Schema) Filter(opt *FilterOption) error {
	i := append(opt.Include, s.NormalizeTableNames(opt.Include)...)
	e := append(opt.Exclude, s.NormalizeTableNames(opt.Exclude)...)

	includes := []*Table{}
	excludes := []*Table{}
	for _, t := range s.Tables {
		li, mi := matchLength(i, t.Name)
		le, me := matchLength(e, t.Name)
		ml := matchLabels(opt.IncludeLabels, t.Labels)
		switch {
		case mi:
			if me && li < le {
				excludes = append(excludes, t)
				continue
			}
			includes = append(includes, t)
		case ml:
			if me {
				excludes = append(excludes, t)
				continue
			}
			includes = append(includes, t)
		case len(opt.Include) == 0 && len(opt.IncludeLabels) == 0:
			if me {
				excludes = append(excludes, t)
				continue
			}
			includes = append(includes, t)
		default:
			excludes = append(excludes, t)
		}
	}

	collects := []*Table{}
	for _, t := range includes {
		ts, _, err := t.CollectTablesAndRelations(opt.Distance, true)
		if err != nil {
			return err
		}
		for _, tt := range ts {
			if !tt.Contains(includes) {
				collects = append(collects, tt)
			}
		}
	}

	for _, t := range excludes {
		if t.Contains(collects) {
			continue
		}
		err := excludeTableFromSchema(t.Name, s)
		if err != nil {
			return errors.Wrap(errors.WithStack(err), fmt.Sprintf("failed to filter table '%s'", t.Name))
		}
	}

	return nil
}

func excludeTableFromSchema(name string, s *Schema) error {
	// Tables
	tables := []*Table{}
	for _, t := range s.Tables {
		if t.Name != name {
			tables = append(tables, t)
		}
		for _, c := range t.Columns {
			// ChildRelations
			childRelations := []*Relation{}
			for _, r := range c.ChildRelations {
				if r.Table.Name != name && r.ParentTable.Name != name {
					childRelations = append(childRelations, r)
				}
			}
			c.ChildRelations = childRelations

			// ParentRelations
			parentRelations := []*Relation{}
			for _, r := range c.ParentRelations {
				if r.Table.Name != name && r.ParentTable.Name != name {
					parentRelations = append(parentRelations, r)
				}
			}
			c.ParentRelations = parentRelations
		}
	}
	s.Tables = tables

	// Relations
	relations := []*Relation{}
	for _, r := range s.Relations {
		if r.Table.Name != name && r.ParentTable.Name != name {
			relations = append(relations, r)
		}
	}
	s.Relations = relations

	return nil
}

func matchLabels(il []string, l Labels) bool {
	for _, ll := range l {
		for _, ill := range il {
			if wildcard.MatchSimple(ill, ll.Name) {
				return true
			}
		}
	}
	return false
}

func matchLength(s []string, e string) (int, bool) {
	for _, v := range s {
		if wildcard.MatchSimple(v, e) {
			return len(strings.ReplaceAll(v, "*", "")), true
		}
	}
	return 0, false
}
