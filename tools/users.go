package tools

import (
	"bufio"
	"context"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetUserInfoInput struct {
	Search string `json:"search,omitempty" jsonschema:"optional username filter (case-insensitive substring match)"`
}

type UserInfo struct {
	Username string   `json:"username"`
	UID      int      `json:"uid"`
	GID      int      `json:"gid"`
	HomeDir  string   `json:"home_directory"`
	Shell    string   `json:"shell"`
	Groups   []string `json:"groups,omitempty"`
}

type GetUserInfoOutput struct {
	Users []UserInfo `json:"users"`
	OutputErrors
}

func parsePasswd() ([]UserInfo, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var users []UserInfo
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 7)
		if len(parts) < 7 {
			continue
		}
		uid, _ := strconv.Atoi(parts[2])
		gid, _ := strconv.Atoi(parts[3])
		users = append(users, UserInfo{
			Username: parts[0],
			UID:      uid,
			GID:      gid,
			HomeDir:  parts[5],
			Shell:    parts[6],
		})
	}
	return users, scanner.Err()
}

func parseGroup() (map[int]string, map[string][]string) {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil, nil
	}
	defer func() { _ = f.Close() }()

	nameByGID := make(map[int]string)
	membersByName := make(map[string][]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 3 {
			continue
		}
		gid, _ := strconv.Atoi(parts[2])
		nameByGID[gid] = parts[0]
		if len(parts) == 4 && parts[3] != "" {
			membersByName[parts[0]] = strings.Split(parts[3], ",")
		}
	}
	return nameByGID, membersByName
}

func buildGroupMembership(
	groupNameByGID map[int]string,
	membersByName map[string][]string,
	username string, primaryGID int,
) []string {
	seen := make(map[string]bool)
	var groups []string
	if name, ok := groupNameByGID[primaryGID]; ok {
		groups = append(groups, name)
		seen[name] = true
	}
	for gname, members := range membersByName {
		if seen[gname] {
			continue
		}
		if slices.Contains(members, username) {
			groups = append(groups, gname)
			seen[gname] = true
		}
	}
	return groups
}

func GatherUserInfo(
	ctx context.Context,
	search string,
) (*GetUserInfoOutput, error) {
	var out GetUserInfoOutput
	var errs ErrList

	users, err := parsePasswd()
	if err != nil {
		errs.Add("passwd", err)
		out.Errors = errs
		return &out, out.Err()
	}

	groupNameByGID, membersByName := parseGroup()

	for _, u := range users {
		if search != "" {
			if !strings.Contains(
				strings.ToLower(u.Username),
				strings.ToLower(search),
			) {
				continue
			}
		}
		if groupNameByGID != nil && membersByName != nil {
			u.Groups = buildGroupMembership(
				groupNameByGID, membersByName,
				u.Username, u.GID,
			)
		}
		if u.Groups == nil {
			u.Groups = []string{}
		}
		out.Users = append(out.Users, u)
	}

	if out.Users == nil {
		out.Users = []UserInfo{}
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetUserInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetUserInfoInput,
) (*mcp.CallToolResult, *GetUserInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_user_info",
		0,
		func(ctx context.Context) (*GetUserInfoOutput, error) {
			return GatherUserInfo(ctx, input.Search)
		},
	)
}
