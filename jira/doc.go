/*
Jira is a program to interact with Jira issues from the Acme editor.
Projects, issues and comments are presented as a virtual read-only filesystem
(using package [io/fs])
which can be browsed in the usual way Acme handles filesystems served
by the host system.

The filesystem root holds project directories.
Within each project are the project's issues, one directory entry per issue.
The filepaths for issues TEST-1, TEST-2, and WEB-27 would be:

	TEST/1
	TEST/2
	WEB/27

Each issue directory has a file named "issue"
holding a textual representation of the issue and a listing of comments.
For example, TEST/1/issue.

Comments are available as numbered files alongside the issue file.
Comment 69 of issue TEST-420 can be accessed at TEST/420/69.

https:developer.atlassian.com/cloud/jira/platform/rest/v2/
https:jira.atlassian.com/rest/api/2/issue/JRA-9
*/
package jira
