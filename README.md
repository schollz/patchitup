<p align="center">
<img
    src="https://raw.githubusercontent.com/schollz/patchitup/master/.github/logo.png?token=AGPyE539EM60WRIJo3VVDBASqzozHprrks5amLWtwA%3D%3D"
    width="260px" border="0" alt="patchitup">
<br>
<a href="https://github.com/schollz/patchitup/releases/latest"><img src="https://img.shields.io/badge/version-0.1.0-brightgreen.svg?style=flat-square" alt="Version"></a>
<img src="https://img.shields.io/badge/coverage-75%25-green.svg?style=flat-square" alt="Code Coverage">
<a href="https://godoc.org/github.com/schollz/patchitup/patchitup"><img src="https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square" alt="Code Coverage"></a>
<a href="https://www.paypal.me/ZackScholl/5.00"><img src="https://img.shields.io/badge/donate-$5-brown.svg?style=flat-square" alt="Donate"></a>
</p>

<p align="center">Backup your file to a cloud server using minimum bandwidth.</p>

*patchitup* is a way to keep the cloud up-to-date through incremental patches. In a nutshell, this is a pure-Golang library and a CLI tool for creating a client+server that exchange incremental gzipped patches to overwrite a remote copy to keep it up-to-date with the client's local file. 

<em><strong>Why?</strong></em> I wrote this program to reduce the bandwidth usage when backing up SQLite databases to a remote server from Raspberry Pis. I have deployed some software on Raspberry Pis that periodically [dumps the database to SQL text](http://www.sqlitetutorial.net/sqlite-dump/). Since Raspberry Pi's can die sometimes, I want to keep their data stored remotely. As the databases can get fairly large, a patch from SQL text will only ever be the changed/new records. *patchitup* allows the client to just send to the cloud only the changed/new records and still maintain the exact copy on the cloud. This can massively reduce bandwidth between the client and the cloud. 

<em><strong>Why not git?</strong></em> While *git* basically  does this already, its not terribly easy to setup a *git* server to support multiple users (though [gitolite](https://github.com/sitaramc/gitolite) does a great job of simplifying the process).  Also, most of the features of *git* are not necessary for my use-case.

# Quickstart

In addition to being a Golang library, the *patchitup* is a server+client. To try it, first install *patchitup* with Go:

```
$ go install -u -v github.com/schollz/patchitup/...
```

Then start a *patchitup* server:

```
$ patchitup -host
Running at http://0.0.0.0:8002
```

Then you can patch a file:

```
$ patchitup -u me -s http://localhost:8002 -f SOMEFILE
2018-02-23 08:56:44 [INFO] patched 2.4 kB (62.8%) to remote 'SOMEFILE' for 'me'
2018-02-23 08:56:44 [INFO] remote server is up-to-date

$ vim SOMEFILE # make some edits

$ patchitup -u me -s http://localhost:8002 -f SOMEFILE
2018-02-23 08:57:40 [INFO] patched 408 B (9.9%) to remote 'SOMEFILE' for 'me'
2018-02-23 08:57:40 [INFO] remote server is up-to-date
```

The first time you patch will basically just send up the gzipped file. Subsequent edits will just send up the patches. The percentage (e.g. `9.9%`) specifies the percentage of the entire file size that is being sent (to get an idea of bandwidth savings). The server also will log bandwidth usage.

Everything stays gzipped on disk.


# How does it work?

_Note:_ *patchitup* does **not** work for binary files (yet).

Why not just do "`diff -u old new > patch && rsync patch your@server:`"? Well, *patchitup* keeps things organized a lot better and uses `gzip` by default to reduce the bandwidth cost even further. Also, in order to patch a remote file you first need a copy of the remote file to create the patch. In *patchitup*, if the local copy of remote file is not available, a local copy of the remote file is reconstructed it in a way that can massively reduce bandwidth (i.e. instead of just downloading the remote file). To reconstruct a local copy of remote file:

1. The client asks the remote server for a hash of every line and its corresponding line number in the remote file. 
2. The client checks to see if any lines are needed (i.e. the set of line hashes that do not exist in the current local file). The client then asks the remote server for the actual lines corresponding to the missing hashes.
3. The client uses these data (the local line hashes, the remote line hashes, and the hash line numbers) to reconstruct a copy of the remote file for doing the patching.

Once the local copy of the remote file is established, a patch is created and gzipped and sent to the server for overwriting the current remote copy. A current remote copy is cached locally so that it need not be reconstructed the next time.

A more detailed flow chart:

<center>
	<img src="https://user-images.githubusercontent.com/6550035/36574282-e0335014-17f9-11e8-92ba-1a474deaae76.png">
</center>

# Roadmap

I would love PRs.

Some ideas I'd like to add:

- [ ] Built-in security (authentication tokens?)
- [ ] Encryption option (to keep data on server private)

# License

MIT

# Thanks

Logo designed by <a rel="nofollow" target="_blank" href="https://www.vecteezy.com">www.Vecteezy.com</a>
