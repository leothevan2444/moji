package client

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leothevan2444/moji/pkg/r18dev/pg/gen"
)

type Client struct {
	pool  *pgxpool.Pool
	query *gen.Queries
}

type Config struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
	MaxConns int32
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	config.ConnConfig.Host = cfg.Host
	config.ConnConfig.Port = cfg.Port
	config.ConnConfig.Database = cfg.Database
	config.ConnConfig.User = cfg.User
	config.ConnConfig.Password = cfg.Password
	config.MaxConns = cfg.MaxConns
	pool, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return nil, err
	}

	client := &Client{pool: pool}
	client.query = gen.New(pool)

	return client, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

type Movie struct {
	ContentID       string
	OtherContentIDs []string
	Code            string
	Title           string
	Comment         string
	ReleaseDate     time.Time
}

func printPgtypeText(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return "none"
}

type FamilyNode struct {
	Video    *gen.GetActressVideosRow
	Parent   *FamilyNode
	Children []*FamilyNode
}

func (n *FamilyNode) otherContentIDs() []string {
	var ids []string
	ids = append(ids, n.Video.DerivedVideo.ContentID)
	for _, child := range n.Children {
		ids = append(ids, child.otherContentIDs()...)
	}
	return ids
}

func insertIntoSubtree(parent *FamilyNode, node *FamilyNode) {
	for _, child := range parent.Children {
		if strings.Contains(child.Video.DerivedVideo.ContentID, node.Video.DerivedVideo.ContentID) {
			node.Children = append(node.Children, child)
			child.Parent = node
			return
		}

		if strings.Contains(node.Video.DerivedVideo.ContentID, child.Video.DerivedVideo.ContentID) {
			insertIntoSubtree(child, node)
			return
		}
	}

	parent.Children = append(parent.Children, node)
	node.Parent = parent
}

func buildFamilies(videos []gen.GetActressVideosRow) []*FamilyNode {
	var families []*FamilyNode

	for i := range videos {
		if !videos[i].DerivedVideo.DvdID.Valid {
			continue
		}
		video := &videos[i]
		newNode := &FamilyNode{Video: video}

		if len(families) == 0 {
			families = append(families, newNode)
			continue
		}

		var inserted bool = false
		for i, root := range families {
			// 新节点应成为当前族的顶级
			if strings.Contains(root.Video.DerivedVideo.ContentID, video.DerivedVideo.ContentID) {
				newNode.Children = append(newNode.Children, root)
				root.Parent = newNode
				families[i] = newNode
				inserted = true
				break
			}

			// 当前族包含新节点
			if strings.Contains(video.DerivedVideo.ContentID, root.Video.DerivedVideo.ContentID) {
				insertIntoSubtree(root, newNode)
				inserted = true
				break
			}
		}
		if !inserted {
			families = append(families, newNode)
		}
	}

	var result []*FamilyNode
	for _, f := range families {
		if len(result) == 0 {
			result = append(result, f)
			continue
		}

		var inserted bool = false
		for i, root := range result {
			if strings.Contains(root.Video.DerivedVideo.ContentID, f.Video.DerivedVideo.ContentID) {
				f.Children = append(f.Children, root)
				root.Parent = f
				result[i] = f
				inserted = true
				break
			}
			if strings.Contains(f.Video.DerivedVideo.ContentID, root.Video.DerivedVideo.ContentID) {
				insertIntoSubtree(root, f)
				inserted = true
				break
			}
		}
		if !inserted {
			result = append(result, f)
		}
	}

	return result
}

func extractMovies(families []*FamilyNode) []Movie {
	var movies []Movie

	for _, node := range families {
		v := node.Video.DerivedVideo
		var otherIDs []string
		for _, child := range node.Children {
			otherIDs = append(otherIDs, child.Video.DerivedVideo.ContentID)
		}
		movies = append(movies, Movie{
			ContentID:       v.ContentID,
			OtherContentIDs: node.otherContentIDs(),
			Code:            printPgtypeText(v.DvdID),
			Title:           printPgtypeText(v.TitleJa),
			Comment:         printPgtypeText(v.CommentJa),
			ReleaseDate:     v.ReleaseDate.Time,
		})
	}
	return movies
}

func (c *Client) GetActressMovies(ctx context.Context, name string) ([]Movie, error) {
	actress_id, err := c.query.GetActressID(ctx, pgtype.Text{String: name, Valid: true})
	if err != nil {
		return nil, err
	}

	rows, err := c.query.GetActressVideos(ctx, actress_id)
	if err != nil {
		return nil, err
	}

	families := buildFamilies(rows)

	movies := extractMovies(families)

	return movies, nil
}

func (c *Client) GetMovie(ctx context.Context, contentID string) (*Movie, error) {
	row, err := c.query.GetVideo(ctx, contentID)
	if err != nil {
		return nil, err
	}

	movie := &Movie{
		ContentID:       row.ContentID,
		OtherContentIDs: []string{},
		Code:            printPgtypeText(row.DvdID),
		Title:           printPgtypeText(row.TitleJa),
		Comment:         printPgtypeText(row.CommentJa),
		ReleaseDate:     row.ReleaseDate.Time,
	}

	return movie, nil
}
