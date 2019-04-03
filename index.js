#!/bin/env node
'use strict';

const express   = require('express');
const request   = require('request');
const path      = require('path');
const exec      = require('child_process').exec;

const app = express();
const gitDir = path.join(__dirname, 'insights-core');
const zipFile = path.join(gitDir, 'insights.zip');

function trimTag(str) {
    if (str.includes('insights-core-')) {
        return str.split('insights-core-')[1]
    }
    if (str.includes('falafel-')) {
        return str.split('falafel-')[1]
    }
    return str;
}

function getBranches(cb) {
    exec('git branch -r', {cwd: gitDir}, (err, stdout, stderr) => {
        if (err) {
            cb(err, null);
            return;
        }
        let branches = stdout.split('\n');
        let filtered = branches.map(b => ({
            name: b.trim().split('origin/')[1],
            fullBranch: b.trim()
        })).filter(b => b.name !== undefined && !b.name.includes('HEAD -> '));
        cb(null, filtered);
    });
}

function getTags(cb) {
    exec('(git checkout master && git pull)  > /dev/null && git --no-pager tag',
         {cwd: gitDir}, (err, stdout, stderr) => {
        if (err) {
            cb(err, null);
            return;
        }
        let tags = stdout.split('\n');
        let filtered = tags.map(t => (
        {
            name: trimTag(t),
            fullTag: t
        }))
        filtered.sort((a,b) => {
            if (a.name > b.name)
                return 1;
            if (a.name < b.name)
                return -1;
            return 0;
        });
        filtered.push({name: 'master', fullTag: 'master'});
        filtered.reverse();
        cb(null, filtered);
    });
}

function createEggFromTag(tag, cb) {
    exec('rm -rf insights.zip && git checkout ' + tag + ' && ./build_client_egg.sh', {cwd: gitDir}, (err, stdout, stderr) => {
        if (err) {
            cb(err, null)
            return;
        }
        cb(null, stdout);
    });
}

app.set('port', 3000);
app.get('/tag/:tag', (req, res) => {
    // create an egg and prep for download
    createEggFromTag(req.params.tag, (err, eggFile) => {
        if (err) {
            res.status(500).send();
            return;
        }
        res.download(zipFile, 'insights-core-' + trimTag(req.params.tag) + '.egg');
    })
});

app.get('/tag', (req, res) => {
    // get list of all eggs by github tag
    getTags((err, data) => {
        if (err) {
            res.status(500).send();
            return;
        }
        res.status(200).send(data);
    })
});

app.get('/branch/:branch', (req, res) => {
    // create an egg and prep for download
    createEggFromTag(req.params.branch, (err, eggFile) => {
        if (err) {
            res.status(500).send();
            return;
        }
        res.download(zipFile, 'insights-core-' + req.params.branch + '.egg');
    })
});

app.get('/branch', (req, res) => {
    // get list of all eggs by github tag
    getBranches((err, data) => {
        if (err) {
            res.status(500).send();
            return;
        }
        res.status(200).send(data);
    })
});

app.get('/', (req, res) => {
    // send main page
    res.sendFile(path.join(__dirname, 'index.html'))
});


app.listen(app.get('port'), () => {
    console.log('egg-hatcher now accepting connections on port %d ...',
                app.get('port'));
});