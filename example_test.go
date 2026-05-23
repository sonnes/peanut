package peanut_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sonnes/peanut"
)

type Tweet struct {
	Author   string
	Text     string
	Likes    int
	Retweets int
	Replies  int
	Views    int
}

type Timeline struct {
	Handle     string
	Name       string
	Followers  int
	Following  int
	TweetCount int
	LikeCount  int
	Tweets     []Tweet
	Response   string
}

type Task = peanut.Task[*Timeline]
type TaskFunc = peanut.TaskFunc[*Timeline]

// TestExample_TimelineRunsEndToEnd is a smoke test ensuring Example_timeline
// actually executes — Go won't run an Example function without an // Output:
// block, and Parallel emits durations we can't pin in a static block.
func TestExample_TimelineRunsEndToEnd(t *testing.T) {
	Example_timeline()
}

func Example_timeline() {
	exec := peanut.New[*Timeline]("timeline",
		peanut.Description("loads and renders a Twitter-style timeline"),
		peanut.Timeout(1*time.Second),
	)

	exec.Add(
		FetchProfile(),
		peanut.Parallel[*Timeline]("fetch").Add(
			FetchTweets(),
			FetchFollowers(),
			FetchFollowing(),
			FetchCounts(),
		),
		RenderTimeline(),
	)

	exec.Use(
		loggingMiddleware(),
	)

	tl := &Timeline{
		Handle: "@gopher",
	}
	if err := exec.Run(context.Background(), tl); err != nil {
		panic(err)
	}
	fmt.Println(tl.Response)
}

func FetchProfile() Task {
	return peanut.MustDefine[*Timeline]("fetch-profile", func(ctx context.Context, s *Timeline) error {
		s.Name = "Go Pher"
		return nil
	}, peanut.Description("Loads display name for the handle"))
}

func FetchTweets() Task {
	return peanut.MustDefine[*Timeline]("fetch-tweets", func(ctx context.Context, s *Timeline) error {
		s.Tweets = []Tweet{
			{Author: s.Handle, Text: "hello, world", Likes: 42, Retweets: 5, Replies: 3, Views: 1024},
			{Author: s.Handle, Text: "shipping peanut today", Likes: 17, Retweets: 2, Replies: 1, Views: 256},
		}
		return nil
	}, peanut.Description("Loads recent tweets for the handle"))
}

func FetchCounts() Task {
	return peanut.MustDefine[*Timeline]("fetch-counts", func(ctx context.Context, s *Timeline) error {
		s.TweetCount = 312
		s.LikeCount = 4096
		return nil
	}, peanut.Description("Loads aggregate tweet and like counts"))
}

func FetchFollowers() Task {
	return peanut.MustDefine[*Timeline]("fetch-followers", func(ctx context.Context, s *Timeline) error {
		s.Followers = 1280
		return nil
	}, peanut.Description("Loads follower count"))
}

func FetchFollowing() Task {
	return peanut.MustDefine[*Timeline]("fetch-following", func(ctx context.Context, s *Timeline) error {
		s.Following = 64
		return nil
	}, peanut.Description("Loads following count"))
}

func RenderTimeline() Task {
	return peanut.MustDefine[*Timeline]("render-timeline", func(ctx context.Context, s *Timeline) error {
		const w = 48
		var b strings.Builder
		bar := "+" + strings.Repeat("-", w-2) + "+\n"
		row := func(text string) {
			fmt.Fprintf(&b, "| %-*s |\n", w-4, text)
		}

		b.WriteString(bar)
		row(fmt.Sprintf("%s  %s", s.Name, s.Handle))
		row(fmt.Sprintf("followers %-6d  following %-6d", s.Followers, s.Following))
		row(fmt.Sprintf("tweets    %-6d  likes     %-6d", s.TweetCount, s.LikeCount))
		b.WriteString(bar)
		for i, t := range s.Tweets {
			row(fmt.Sprintf("%s", t.Author))
			row(t.Text)
			row(fmt.Sprintf("%d likes  %d rt  %d replies  %d views", t.Likes, t.Retweets, t.Replies, t.Views))
			if i < len(s.Tweets)-1 {
				b.WriteString("|" + strings.Repeat(".", w-2) + "|\n")
			}
		}
		b.WriteString(bar)
		s.Response = b.String()
		return nil
	}, peanut.Description("Composes the ASCII timeline into Response"))
}

func loggingMiddleware() peanut.Middleware[*Timeline] {
	return func(next Task) Task {
		def := next.Def()
		return peanut.WithDef(def, func(ctx context.Context, s *Timeline) error {
			start := time.Now()
			err := next.Run(ctx, s)
			fmt.Printf("%s done in %s\n", def.Name, time.Since(start))
			return err
		})
	}
}

