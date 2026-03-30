# file-storage
## Intro
Building a (good) web application almost always involves handling "large" files of some kind - whether its static images and videos for a marketing site, or use generated content like profile pictures and video uploads, it always seems to come up.

We'll cover strategies for handling files that are kilobytes, megabytes, or even *gigabytes* in size, as opposed to small structured data you might store in a traditional database (integers, booleans, and simple strings).

### Learning goals
* Understand what "large" files are and how they differ from "small" structured data
* Build an app that uses AWS S3 and Go to store and serve assets
* Learn how to manage files on a "normal" (non-s3) filesystem based application
* Learn how to store and serve assets at scale using serverless solutions, like AWS S3
* Learn how to stream video and to keep data usage low and improve performance

### Tubely
We'll be building "Tubely", a SaaS product that helps YouTubers manage their video assets. It allows users to upload, store, serve, add metadata to, and version their video files. It will also allow them to manage thumbnails, titles, and other video metadata.

## Large Files
What are "large files"?

Small structured data is stuff that is usually stored in a relational database like Postgres or MySQL. Something like:
* `user_id` (integer)
* `is_active` (boolean)
* `email` (string)

**Large files**, or large assets are giant globs of data encoded in a specific file format and measured in kilo, mega, or gigabytes. As a simple rule:
* If it makes sense to go into an excel spreadsheet, it probably belongs in a traditional database.
* If it would normally be stored on your HDD as its own file, its probably a "large file".

Large files are interesting because:
1. They're large in size and thus are more performance intensive.
2. They're usually accessed frequently, and this combined with their size can quickly lead to performance bottlenecks.

## Database
Tubely's architecture is simple. We're using:
* Golang to write the application.
* SQLite as a database. SQLite is a traditional relational database that works out of a single flat file, meaning it doesn't require a seperate server process to run.
* Later, we'll also use the filesystem and S3 to store large files.

### Assignment 
1. With the server running, create a Tubely account by entering the following email and password and clicking "sign up".

We'll use these credentials for the entire course!

2. Install SQLite 3. The application is already set up to use SQLite, I just want you to be able to use the CLI to manually inspect the database.

#### linux
sudo apt update
sudo apt install sqlite3

#### mac
brew update
brew install sqlite3

3. Run sqlite3 `tubely.db` to open the database file (that should have been created automatically by the server when you started it). Run a `select * from users;` to see the users table. You should see yourself in there!

4. Type `.exit` to exit the SQLite CLI.

## Videos
The two main entities in Tubely are `videos` and `users`. A `user` can have many `videos`, and a `video` belongs to a single `user`.

"Videos" have 3 things to worry about:
* Metadata: The title, description, and other informatin about the video.
* Thumbnail: An image that represents the video.
* Video: The actual video file.

Tubely allows users to create a "new draft" - which creates a new video record in the database containing metadata only. Thumbnails and video files are uploaded separately after the draft is created.

## Encoding
As you know, we use a SQLite database to power the majority of the web app. SQLite is a traditional relational database that works out of a single flat file, meaning it doesn't need a separate server process to run.

Let's talk about the elephant in the room: Our current solution for video thumbnails (storing the media in-memory) is a terrible solution. If the server is restarted, all the thumbnails are lost!

But we can't store an image in a SQLite column?... Right?

To do so, we can actually encode the image as a [base64](https://en.wikipedia.org/wiki/Base64) string and shove the whole thing into a text column in SQLite. Base64 is just a way to encode binary (raw) data as text. It's not the most efficient way to do it, but it will work for now.

### Assignment
Update the code to store the image data in the thumbnail_url column in the database.

1. Use base64.StdEncoding.EncodeToString from the encoding/base64 package to convert the image data to a base64 string.
2. Create a data URL with the media type and base64 encoded image data. The format is:
    ```
    data:<media-type>;base64,<data>
    ```
3. Store the URL in the `thumbnail_url` column in the database.
4. Because the `thumbnail_url` has all the data we need, delete the global thumbnail map and the `GET` route for thumbnails.
5. Restart the server and re-upload the `boots-image-horizontal.png` thumbnail image to ensure it's working.

## Using the Filesystem
Great, now we're using `base64` strings in our SQLite database to store images. Let's talk about why that actually kinda sucks.
1. **CPU performance**: Base64 encoding is an expensive CPU-intensive operation. If we have a lot of uploads (I mean, we're planning on being a successful company right?), we'll see some scaling issues.
2. **Storage costs** Base64 encoding bloats the size of the image data. We're using more disk space than we need to, which again, is expensive and slow.
3. **Database performance**: Databases (especially relational databases) are optimized for small, strucutred data, not giant blobs of binary. It will impact query performance in a non-trivial way.
4. **Caching**: Base64 encoded images aren't as cache friendly as raw files, meaning slower load times and higher bandwidth costs.

It's usually a bad idea to store large binary blobs in a database, there are exceptions, but they are rare. So what's the solution? **Store the files on the file system**. File systems are optimized for storing and serving files, and they do it well.

### Assignment
Let's update our handler to store the files on the file system. We'll save uploaded files to the `/assets` directory on disk.

1. Instead of encoding to base64, update the handler to save the bytes to a file at the path `/assets/<videoID>.<file_extension>`.
    1. Use the `Content-Type` header to determine the file extension.
    2. Use the `videoID` to create a unique file path. `filepath.Join` and `cfg.assetsRoot` will be helpful here.
    3. Use `os.Create` to create the new file
    4. Copy the contents from the `multipart.File` to the new file on disk using `io.Copy`
2. Update the `thumbnail_url`. Notice that in `main.go` we have a file server that serves files from the `/assets` directory. The URL for the thumbnail should now be:
```
http://localhost:<port>/assets/<videoID>.<file_extension>
```

3. Restart the server and re-upload the boots-image-horizontal.png thumbnail image to ensure it's working. You should see it in the UI as well as a copy in the /assets directory.

Run and submit the CLI tests.
