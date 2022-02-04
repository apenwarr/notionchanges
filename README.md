## Unofficial recent changes lister for Notion.so sites

Notion.so is a nice document database, but its notifications system is very
bad:

- By default it only shows you pages you've subscribed to.

- If you subscribe to too many pages, or look at the "All" tab, the list of
  changes is very cluttered.

- If someone works all day making 100 changes to a single page, and five
  other pages get one simple change each during the day, the 100
  notifications drown out the five notifications so you'll never see them.

- The change view is in a tiny box that doesn't make full use of your
  browser window.

- It's hard to tell which changes you have or haven't already looked at.

- It's hard to tell which pages have changed recently and which haven't.

This app, which uses the "unofficial" Notion.so api client from
https://github.com/kjk/notionapi, tries to resolve all these issues in the
simplest possible way:

- In a single view, it shows each (somewhat recently modified) page in your
  notion database exactly once, along with who edited it most recently and
  which day it was edited.

- Each page link contains a revision indicator so that your browser can tell
  if you've visited that revision before or not. The link colour changes if
  you have. This makes it easy to tell, at a glance, which revisions you've
  read or not read, and which things have changed since the last time you
  opened a page.

- It's a totally dumb, flat, no-javascript html view that expands to fill
  the space available on your screen, and loads very fast.

- That's it!


## Configuring notionchanges

You need to provide two files, which must be in the current working
directory at the time you start the `notionchanges` binary:

- `notion.key`: a text file containing the `token_v2` cookie extracted from
  your notion session. The app uses this to access the notion API using your
  personal rights. You might need to refresh this file occasionally if you
  get logged out.

- `space.id`: a text file containing the notion id of the "space" that forms the root of your
  notion database. You can find this by using Chrome's debugger to inspect
  the response to a `loadCachedPageChunk` command in notion. (It generates at
  least one of these every time you load notion.) Look inside the returned
  `recordMap` object, then in the `space` object, and copy the uuid in
  there. [TODO: if none is configured, offer a list of spaces to choose
  from.]


## Running notionchanges

To run the program, use these steps:

- `git clone https://github.com/apenwarr/notionchanges`
- `go run .`
- Visit http://localhost:8187/


## How do I control which users can see the page?

It's not a good idea to publish the notionchanges view directly on the
Internet, since it has no authentication. You can put it behind a proxy or
control access using a VPN such as [Tailscale](https://tailscale.com/).


## How can I contribute?

There are surely many ways to improve this project.

Unfortunately I don't have time to fix most bugs or add new features. Sorry!
But you can fork this repo and submit pull requests and I'll do my best to
look at them and merge them if they're a good fit.
