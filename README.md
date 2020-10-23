# cache-clobber

Renames your js/css files based on their contents with a hash at the end. 
Ensures your html matches the new names.

`<script src="bloat.js">` => `<script src="bloat-cc2530066345.js">`

Does a recursive search in `.html` files for references in `src` and `href` attributes. 

## Why?

Your browser will download your js/css files once and store them into a cache based on their file name. Next visit, it will not download the file names it has cached and use its local copies instead. 

You may upload a new version, and your website will not display your cool new features to the users!

cache-clobber solves this by appending a hash `-ccXXXXXXX`, dependent upon the file's contents, to the file name so every change renders. This is a method of "cache busting".

## Notes

- cache-clobber decides to not use a query parameter to cache bust, since CDNs and proxies may ignore query parameters.
- Kept in one `go run`able file, if you do not like keeping mysterious binaries in your repos.
- Created after not wanting to bother with gulp, grunt, or other configs.