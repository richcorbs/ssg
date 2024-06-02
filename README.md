# Simple Static Site Generator

This is a simple static site generator built in golang. I built it for my needs which are decidedly simple.

## Features

1. VanJS built-in for dynamic content.
2. Pico CSS support.
3. Layout files for all your content. There is a default if you only have one layout.
4. Support for HTML snippets/fragments.
5. Support for HTML and Markdown content.
6. Support for assets like images, javascript, css, etc.
7. Hot reload development experience.
8. One executable for build, dev, and deploy.

## Requirements

1. Download the sssg release for your platform.
2. Put your HTML (.html) and markdown (.md) pages in the ./src/pages directory. Nested directories are ok. `.html` and `.md` files get wrapped in the layout so they don't have to been complete html docs.
3. Customize the layout in `./src/layouts/layout.html`. This way you have one layout and all of your pages get wrapped in the same layout. Be sure to have `__CONTENT__` somewhere in your layout.
4. In the layout file customize the link to your chosen CSS files. We've chosen Pico CSS to include in the init files.
5. Put your static content (images, .js, .css, etc) in the ./src/assets directory and then link to the files like you normally would (/assets/js/whatever.js). They will be copied straight across to `./dist/assets` during the build process.
6. The required directory structure is like this.

```
my-site
├── dist // build will put your built site here, and deploy will deploy from here
└── src // build is expecting your source files to be here
    └── assets // the contents of this directory will be copied straight across to /dist/assets
    │   ├── css
    │   │   ├── pico.colors.min.css
    │   │   ├── pico.min.css
    │   │   └── styles.css
    │   ├── images
    │   │   └── logo.png
    │   └── js
    │       └── app.js
    ├── snippets
    │   └── Test.html
    ├── layouts
    │   ├── default.html
    │   └── layout.html
    └── pages
        ├── about.html
        ├── index.html
        └── markdown.md
```

7. You don't have to create `./dist`. The build process will create it for you.
8. The `init` feature will create the `./src` directory and all of its contents for you.
## To Use SSSG

- Download the sssg release for your platform.

- To initialize your project with a minimal project skeleton:
  - Create a project directory `~/code/my-site`.
  - Run `cd ~/code/my-site`.
  - Run `sssg init` to create the project skeleton in `./src`. This is important because this is where the build expects your source files to be.

- For development run `sssg dev`. This will:
  - Build the site.
  - Serve the site on port 8080.
  - Watch for file changes in the `./src` directory and then rebuild pages/content as needed.
  - Hot reload the browser after the site rebuilds when there is a file change.

- To build run `sssg build`. This will put the rendered content in `./dist`.

- To deploy:
  - Configure private key SSH access to your server. Add your key to the ssh agent if you have a password-protected SSH key.
  - Configure .env with DEPLOY_HOST, DEPLOY_PORT, DEPLOY_DIR
  - Run `sssg deploy`. This will copy the contents of `./dist` to `DEPLOY_DIR` on `DEPLOY_HOST:DEPLOY_PORT`.
  - If you are using ssss the updated content will be available at your site's URL.

## The Future of SSSG

- DONE. I may add support for dumb HTML snippets/fragments but maybe not.
