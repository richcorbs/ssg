#!/bin/bash

SRC_DIR="./src"
DIST_DIR="./dist"
CONTENT_PLACEHOLDER="__CONTENT__"
LAYOUT=$(<"$SRC_DIR/layouts/layout.html")
        
# Function to process a single file (HTML, Markdown, JS, or CSS)
process_file() {
    local input_file="$1"
    
    # Determine the file extension
    local extension="${input_file##*.}"
    
    # Remove the "pages" or "public" top-level directory from the output path
    local output_file="$DIST_DIR/${input_file#$SRC_DIR/}"
    
    # Exclude "public" directory from the structure in ./dist
    if [[ "$input_file" == "$SRC_DIR/public"* ]]; then
        output_file="$DIST_DIR/${input_file#$SRC_DIR/public/}"
    fi
    
    # Exclude "pages" directory from the structure in ./dist
    if [[ "$input_file" == "$SRC_DIR/pages"* ]]; then
        output_file="$DIST_DIR/${input_file#$SRC_DIR/pages/}"
    fi
    
    # Change the file extension to ".html" for HTML and Markdown files
    if [[ "$extension" == "html" || "$extension" == "md" ]]; then
        output_file="${output_file%.$extension}.html"
    fi
    
    # Ensure the output directory exists
    mkdir -p "$(dirname "$output_file")"
    
    # Read the content of the file
    content=$(<"$input_file")
    
    # For HTML files, wrap the content in the layout
    if [ "$extension" == "html" ]; then
        # Replace the content placeholder with the actual content
        output_content="${LAYOUT//$CONTENT_PLACEHOLDER/$content}"
    elif [ "$extension" == "md" ]; then
        # Convert Markdown to HTML using pandoc
        converted_content=$(pandoc "$input_file")
        
        # Replace the content placeholder with the converted Markdown content
        output_content="${LAYOUT//$CONTENT_PLACEHOLDER/$converted_content}"
    else
        # For JS and CSS files, use the content directly without wrapping
        output_content="$content"
    fi
    
    # Save to the output file
    echo "$output_content" > "$output_file"
    echo "  Processed: $input_file -> $output_file"
}

# Function to process all files in a directory
process_directory() {
    local dir="$1"
    
    # Loop through files in the directory
    find "$dir" -type f | while read -r file; do
        process_file "$file"
    done
}

# Function to clear the dist folder
clear_dist() {
    echo "Clearing the dist folder..."
    rm -rf "$DIST_DIR"/*
}

# Function to deploy the generated pages/site
deploy() {
    echo "Deploying to your hosting service..."
    # Add your deployment logic here
    # For example, you can use rsync, scp, or any other method to upload to a server
}

# Clear the dist folder
clear_dist

# Process files in the public directory
echo "Processing /public"
process_directory "$SRC_DIR/public"

# Process files in the pages directory
echo "Processing /pages"
process_directory "$SRC_DIR/pages"

# Deploy option
if [ "$1" == "--deploy" ]; then
    deploy
fi
