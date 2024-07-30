package main

import (
	"strings"
	"testing"
)

func TestParseIssue(t *testing.T) {
	r := strings.NewReader(`Title: VizOne asset metadata fetched as XML, can get JSON
State: opened
Assignee:
Created: 2022-08-04 03:04:01.233 +0000 UTC
URL: https://gitlab.skyracing.cloud/sol1-software/ai/job-service/-/issues/22

Woody helpfully showed us that we can get asset metadata as JSON instead of atom/xml! Cool!

Right now we're importing a package xml2js then doing tedious object attribute traversal.

const meta = {
  title: data['atom:entry']['atom:title'] ? data['atom:entry']['atom:title'][0] : null,
  video: data['atom:entry']['media:group'] ? data['atom:entry']['media:group'][0]['media:content'][0]['$'] : null,
  updated: data['atom:entry']['atom:updated'] ? data['atom:entry']['atom:updated'][0] : null,
  author: data['atom:entry']['atom:author'] ? data['atom:entry']['atom:author'][0]['atom:name'][0] : null,
  mediastatus: data['atom:entry']['mam:mediastatus'] ? data['atom:entry']['mam:mediastatus'][0] : null,`)
	parseIssue(r)
}
