package bw

import "net/url"

type ListObjectsOption func(args *[]string, q *url.Values)
type ListObjectsOptionGenerator func(id string) ListObjectsOption

func WithCollectionID(id string) ListObjectsOption {
	return func(args *[]string, q *url.Values) {
		if q != nil {
			q.Set("collectionId", id)
		} else {
			*args = append(*args, "--collectionid", id)
		}
	}
}

func WithFolderID(id string) ListObjectsOption {
	return func(args *[]string, q *url.Values) {
		if q != nil {
			q.Set("folderid", id)
		} else {
			*args = append(*args, "--folderid", id)
		}
	}
}

func WithOrganizationID(id string) ListObjectsOption {
	return func(args *[]string, q *url.Values) {
		if q != nil {
			q.Set("organizationId", id)
		} else {
			*args = append(*args, "--organizationid", id)
		}
	}
}

func WithSearch(search string) ListObjectsOption {
	return func(args *[]string, q *url.Values) {
		if q != nil {
			q.Set("search", search)
		} else {
			*args = append(*args, "--search", search)
		}
	}
}

func WithUrl(uri string) ListObjectsOption {
	return func(args *[]string, q *url.Values) {
		if q != nil {
			q.Set("url", uri)
		} else {
			*args = append(*args, "--url", uri)
		}
	}
}
