package sqldocument

import (
	"errors"
	"fmt"
)

var CycleError = errors.New("Detected a dependency cycle")

type NotFoundError struct {
	Name string
}

func (n NotFoundError) Error() string {
	return fmt.Sprintf("Name not found: %s", n.Name)
}

func TopologicalSort(input []Create) (output []Create, errpos Pos, err error) {
	visiting := make([]bool, len(input))
	visited := make([]bool, len(input))

	// map of declared name to the index of the SourceFile that declares it...
	declaredToIdx := make(map[string]int)
	for i, f := range input {
		declaredToIdx[f.QuotedName.Value] = i
	}

	var visit func(i int) (Pos, error)

	visit = func(i int) (Pos, error) {
		if visited[i] {
			return Pos{}, nil
		}
		if visiting[i] {
			panic("This should always be caught by an if-test further down")
		}
		visiting[i] = true

		for _, use := range input[i].DependsOn {
			dep, ok := declaredToIdx[use.Value]
			if !ok {
				return use.Pos, NotFoundError{Name: use.Value}
			}

			if visiting[dep] {
				return use.Pos, CycleError
			}
			if errposInner, errInner := visit(dep); errInner != nil {
				return errposInner, errInner
			}
		}

		visiting[i] = false
		visited[i] = true
		output = append(output, input[i])
		return Pos{}, nil
	}

	for i := range input {
		errpos, err = visit(i)
		if err != nil {
			output = nil
			return
		}
	}

	return
}
