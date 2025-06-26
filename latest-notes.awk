# For example
#	goawk -f latest-notes.awk release_note_*.md | gh release create -d --notes-file - -t $(VERSION) $(VERSION) $(wildcard $(NAME)-$(VERSION)-*.zip)

match($0,/^v[0-9]+\.[0-9]+\.[0-9]+$/) > 0 {
    flag = ++f[FILENAME]
    if ( flag == 1 ) {
        version = substr($0,RSTART,RLENGTH)
        printf "\n### Changes in %s ",version
        if (FILENAME ~ /ja/) {
            print "(Japanese)"
        } else {
            print "(English)"
        }
    }
}

f[FILENAME]==1 && /^$/{
    section++
}

(f[FILENAME]==1 && section %2 == 1 ){
    print
}

#gist https://gist.github.com/hymkor/b5ee45313143a838e72b2ed2314ca8e3
