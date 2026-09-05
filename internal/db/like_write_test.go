package db

import (
	"context"
	"testing"
	"time"
)

func scratchThread(t *testing.T, d *DB) int64 {
	t.Helper()
	ctx := context.Background()
	article := scratchArticle(t, d)
	var id int64
	err := d.pool.QueryRow(ctx, `
INSERT INTO web_forumthread (article_id, name, description, created_at, updated_at, is_pinned, is_locked)
VALUES ($1, 'Probe Like Thread', '', now(), now(), false, false)
RETURNING id`, article).Scan(&id)
	if err != nil {
		t.Fatalf("insert scratch thread err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(),
			`DELETE FROM web_forumthread WHERE id = $1`, id); err != nil {
			t.Errorf("clean up scratch thread err = %v, want nil", err)
		}
	})
	return id
}

func scratchPost(t *testing.T, d *DB, threadID int64, name string) int64 {
	t.Helper()
	var id int64
	err := d.pool.QueryRow(context.Background(), `
INSERT INTO web_forumpost (thread_id, name, created_at, updated_at)
VALUES ($1, $2, now(), now())
RETURNING id`, threadID, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert scratch post err = %v, want nil", err)
	}
	t.Cleanup(func() {
		if _, err := d.pool.Exec(context.Background(),
			`DELETE FROM web_forumpost WHERE id = $1`, id); err != nil {
			t.Errorf("clean up scratch post err = %v, want nil", err)
		}
	})
	return id
}

func TestLikePostCountsOnce(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	post := scratchPost(t, d, scratchThread(t, d), "probe")
	user := scratchUser(t, d, "probe-like-a")

	for i := 0; i < 2; i++ {
		if _, err := d.LikePost(ctx, post, user, time.Now().UTC()); err != nil {
			t.Fatalf("LikePost() err = %v, want nil", err)
		}
	}
	got, err := d.PostLikeCount(ctx, post)
	if err != nil {
		t.Fatalf("PostLikeCount() err = %v, want nil", err)
	}
	if got != 1 {
		t.Errorf("PostLikeCount() = %d, want 1", got)
	}
}

func TestUnlikePostTakesTheLikeBack(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	post := scratchPost(t, d, scratchThread(t, d), "probe")
	user := scratchUser(t, d, "probe-like-b")

	if _, err := d.LikePost(ctx, post, user, time.Now().UTC()); err != nil {
		t.Fatalf("LikePost() err = %v, want nil", err)
	}
	if err := d.UnlikePost(ctx, post, user); err != nil {
		t.Fatalf("UnlikePost() err = %v, want nil", err)
	}
	got, err := d.PostLikeCount(ctx, post)
	if err != nil {
		t.Fatalf("PostLikeCount() err = %v, want nil", err)
	}
	if got != 0 {
		t.Errorf("PostLikeCount() = %d, want 0", got)
	}
}

func TestUnlikePostThatWasNeverLiked(t *testing.T) {
	d := writeTestDB(t)
	post := scratchPost(t, d, scratchThread(t, d), "probe")
	user := scratchUser(t, d, "probe-like-c")

	if err := d.UnlikePost(context.Background(), post, user); err != nil {
		t.Errorf("UnlikePost() err = %v, want nil", err)
	}
}

func TestPostLikeCountsAnswerEveryPostAtOnce(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	thread := scratchThread(t, d)
	first := scratchPost(t, d, thread, "probe one")
	second := scratchPost(t, d, thread, "probe two")
	third := scratchPost(t, d, thread, "probe three")
	users := []int64{
		scratchUser(t, d, "probe-like-d"),
		scratchUser(t, d, "probe-like-e"),
	}

	for _, u := range users {
		if _, err := d.LikePost(ctx, first, u, time.Now().UTC()); err != nil {
			t.Fatalf("LikePost() err = %v, want nil", err)
		}
	}
	if _, err := d.LikePost(ctx, second, users[0], time.Now().UTC()); err != nil {
		t.Fatalf("LikePost() err = %v, want nil", err)
	}

	got, err := d.PostLikeCounts(ctx, []int64{first, second, third})
	if err != nil {
		t.Fatalf("PostLikeCounts() err = %v, want nil", err)
	}
	if got[first] != 2 {
		t.Errorf("PostLikeCounts()[first] = %d, want 2", got[first])
	}
	if got[second] != 1 {
		t.Errorf("PostLikeCounts()[second] = %d, want 1", got[second])
	}
	if _, ok := got[third]; ok {
		t.Errorf("PostLikeCounts() holds the unliked post, want it absent")
	}
}

func TestPostsLikedByNamesOnlyTheReadersOwn(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	thread := scratchThread(t, d)
	mine := scratchPost(t, d, thread, "probe mine")
	theirs := scratchPost(t, d, thread, "probe theirs")
	me := scratchUser(t, d, "probe-like-f")
	other := scratchUser(t, d, "probe-like-g")

	if _, err := d.LikePost(ctx, mine, me, time.Now().UTC()); err != nil {
		t.Fatalf("LikePost() err = %v, want nil", err)
	}
	if _, err := d.LikePost(ctx, theirs, other, time.Now().UTC()); err != nil {
		t.Fatalf("LikePost() err = %v, want nil", err)
	}

	got, err := d.PostsLikedBy(ctx, me, []int64{mine, theirs})
	if err != nil {
		t.Fatalf("PostsLikedBy() err = %v, want nil", err)
	}
	if !got[mine] {
		t.Errorf("PostsLikedBy()[mine] = false, want true")
	}
	if got[theirs] {
		t.Errorf("PostsLikedBy()[theirs] = true, want false")
	}
}

func TestPostLikersComeBackNewestFirstInAWindow(t *testing.T) {
	d := writeTestDB(t)
	ctx := context.Background()
	post := scratchPost(t, d, scratchThread(t, d), "probe")
	at := time.Now().UTC()

	names := []string{"probe-like-h", "probe-like-i", "probe-like-j"}
	ids := make([]int64, 0, len(names))
	for i, name := range names {
		id := scratchUser(t, d, name)
		ids = append(ids, id)
		if _, err := d.LikePost(ctx, post, id, at.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("LikePost() err = %v, want nil", err)
		}
	}

	first, err := d.PostLikers(ctx, post, 0, 2)
	if err != nil {
		t.Fatalf("PostLikers() err = %v, want nil", err)
	}
	if len(first) != 2 {
		t.Fatalf("len(PostLikers(0, 2)) = %d, want 2", len(first))
	}
	if first[0].ID != ids[2] {
		t.Errorf("PostLikers()[0].ID = %d, want %d", first[0].ID, ids[2])
	}
	if first[1].ID != ids[1] {
		t.Errorf("PostLikers()[1].ID = %d, want %d", first[1].ID, ids[1])
	}

	second, err := d.PostLikers(ctx, post, 2, 2)
	if err != nil {
		t.Fatalf("PostLikers() err = %v, want nil", err)
	}
	if len(second) != 1 {
		t.Fatalf("len(PostLikers(2, 2)) = %d, want 1", len(second))
	}
	if second[0].ID != ids[0] {
		t.Errorf("PostLikers(2, 2)[0].ID = %d, want %d", second[0].ID, ids[0])
	}
}
