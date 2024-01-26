# Simple Static Site Generator

This is a simple static site generator built in bash. I built it for my needs which are, well, simple.

## Features

1. VanJS built-in for dynamic content.
2. Tailwind CSS support via https://curlwind.com.
3. Single layout file for all your content.
4. Support for HTML and Markdown content.
5. Hot reload development experience.
6. Simple. ~100 lines of bash script.

## Requirements

1. Bash
2. `brew install fswatch` or similar for your OS. I'm on MacOS and so that's what's available to me.
3. `npm i -g browser-sync`
4. Put your HTML (.html) and markdown (.md) pages in the ./src/pages directory. Nested directories are ok. `.html` and `.md` files get wrapped in the layout so they don't have to been complete html docs.
5. Customize the layout in `./src/layouts/layout.html`. This way you have one layout and all of your pages get wrapped in the same layout. Be sure to have `__CONTENT__` somewhere in your layout.
6. In the layout file customize the link to `https://cdn.curlwind.com` with the particular Tailwind CSS classes you need. You can learn more at https://curlwind.com.
7. Put your public content (.js, .css) in the ./src/public directory and then link to the files like you normally would.

## To Use SSSG

- For development run `./dev.sh`. This will give you hot reload anytime you change something in the `./src` directory.

- To build run `./build.sh`.

- To deploy run `./build.sh --deploy`. You'll need to add code to handle the deploy in the `deploy()` function in `build.sh`

That's it.

## The Future of SSSG

- I may add support for components/partials but maybe not.

