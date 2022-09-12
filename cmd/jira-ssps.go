package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/andygrunwald/go-jira"
	"github.com/samber/lo"
)

type stats struct {
	env Environment

	client  *jira.Client
	board   *jira.Board
	sprints []jira.Sprint
}

func main() {
	s := &stats{
		env: getEnvironment(),
	}

	s.getClient()
	s.getBoard()
	s.getSortedSprints()
	s.printSprintsIssuesByAssignee()
}

func (s *stats) getClient() {
	tp := jira.BearerAuthTransport{
		Token: s.env["API_KEY"],
	}

	client, err := jira.NewClient(tp.Client(), s.env["JIRA_URL"])
	if err != nil {
		panic(err)
	}

	s.client = client
}

func (s *stats) getBoard() {
	boards, _, err := s.client.Board.GetAllBoards(&jira.BoardListOptions{
		Name:           s.env["BOARD_NAME"],
		ProjectKeyOrID: s.env["PROJECT_KEY"],
	})
	if err != nil {
		panic(err)
	}
	if boards.Total == 0 {
		panic("No boards found")
	}

	s.board = &boards.Values[0]
}

func (s *stats) getSortedSprints() {
	s.getSprints()

	s.sprints = lo.Filter(s.sprints, func(sprint jira.Sprint, _ int) bool {
		return strings.Contains(sprint.Name, s.env["SPRINT_PREFIX"])
	})

	sort.Slice(s.sprints, func(i, j int) bool {
		return s.sprints[i].StartDate.UTC().After(s.sprints[j].StartDate.UTC())
	})
}

func (s *stats) getSprints() {
	currentPage := 0
	itemsPerPage := 50

	var allSprints []jira.Sprint

	for currentPage != -1 {
		sprints := s.fetchSprints(itemsPerPage, currentPage)

		allSprints = append(allSprints, sprints.Values...)
		if sprints.IsLast {
			currentPage = -1
		} else {
			currentPage++
		}
	}

	s.sprints = allSprints
}

func (s *stats) fetchSprints(itemsPerPage int, currentPage int) *jira.SprintsList {
	sprints, _, err := s.client.Board.GetAllSprintsWithOptions(
		s.board.ID,
		&jira.GetAllSprintsOptions{
			State: "closed",
			SearchOptions: jira.SearchOptions{
				MaxResults: itemsPerPage,
				StartAt:    currentPage * itemsPerPage,
			},
		},
	)
	if err != nil {
		panic(err)
	}

	return sprints
}

func (s *stats) printSprintsIssuesByAssignee() {
	maxI, err := strconv.Atoi(s.env["NUMBER_OF_SPRINTS_TO_ANALYZE"])
	if err != nil {
		panic(err)
	}

	sprintsStoryPoints := make([]int, maxI)

	for _, sprint := range s.sprints {
		if maxI == 0 {
			break
		}
		sprintsStoryPoints[maxI-1] = s.getSprintStoryPoints(&sprint)
		maxI--
	}

	fmt.Println(sprintsStoryPoints)
}

func (s *stats) getSprintStoryPoints(sprint *jira.Sprint) int {
	sprintStoryPoints := 0

	issues := s.getSprintIssuesByAssignee(sprint)
	for _, issue := range issues {
		fields, _, _ := s.client.Issue.GetCustomFields(issue.ID)
		storyPointsField := fields["customfield_12310243"]
		if storyPointsField != "<nil>" {
			storyPoints, err := strconv.Atoi(storyPointsField)
			if err != nil {
				panic(err)
			}
			sprintStoryPoints = sprintStoryPoints + storyPoints
		}
	}

	return sprintStoryPoints
}

func (s *stats) getSprintIssuesByAssignee(sprint *jira.Sprint) []jira.Issue {
	issues, _, err := s.client.Sprint.GetIssuesForSprint(sprint.ID)
	if err != nil {
		panic(err)
	}

	return lo.Filter(issues, func(issue jira.Issue, _ int) bool {
		return issue.Fields != nil &&
			issue.Fields.Assignee != nil &&
			issue.Fields.Assignee.EmailAddress == s.env["ASSIGNEE_EMAIL_ADDRESS"]
	})
}
