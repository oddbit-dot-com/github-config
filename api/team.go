package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ensureTeams provisions teams and their memberships
func (o *Organization) ensureTeams(ctx *pulumi.Context, provider *github.Provider) error {
	for teamKey, teamConfig := range o.Teams {
		// Build team args — use Settings if provided, otherwise start with empty args
		var args *github.TeamArgs
		if teamConfig.Settings != nil {
			settingsCopy := *teamConfig.Settings
			args = &settingsCopy
		} else {
			args = &github.TeamArgs{}
		}

		// Use map key as team name if Settings.Name is not set
		if args.Name == nil {
			args.Name = pulumi.String(teamKey)
		}

		// Create team
		resourceName := fmt.Sprintf("github_team.%s.%s", helpers.Slugify(o.Name), helpers.Slugify(teamKey))
		team, err := github.NewTeam(ctx, resourceName, args, pulumi.Provider(provider))
		if err != nil {
			return fmt.Errorf("failed to create team %s (%s): %w", o.Name, teamKey, err)
		}

		// Create team members if any are specified
		if len(teamConfig.Members) > 0 {
			members := make(github.TeamMembersMemberArray, 0, len(teamConfig.Members))
			for username, role := range teamConfig.Members {
				members = append(members, &github.TeamMembersMemberArgs{
					Username: pulumi.String(username),
					Role:     pulumi.String(role),
				})
			}

			membersResourceName := fmt.Sprintf("github_team_members.%s.%s", helpers.Slugify(o.Name), helpers.Slugify(teamKey))
			_, err = github.NewTeamMembers(ctx, membersResourceName, &github.TeamMembersArgs{
				TeamId:  team.ID(),
				Members: members,
			}, pulumi.Provider(provider))
			if err != nil {
				return fmt.Errorf("failed to create team members for %s (%s): %w", o.Name, teamKey, err)
			}
		}
	}
	return nil
}
