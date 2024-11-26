#!/bin/bash

mkdir -p samples

image_urls=(
    "https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/boots-image-horizontal.png"
    "https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/boots-image-vertical.png"
    "https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/boots-video-horizontal.mp4"
    "https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/boots-video-vertical.mp4"
    "https://storage.googleapis.com/qvault-webapp-dynamic-assets/course_assets/is-bootdev-for-you.pdf"
)

for url in "${image_urls[@]}"; do
  file_name=$(basename "$url")
  curl -sSfL -o "samples/$file_name" "$url"
done
