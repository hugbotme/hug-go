#!/usr/bin/env ruby
# encoding: utf-8

require 'rubygems'
require 'bundler/setup'
require 'redic'
require 'github-trending'

def canonicalize owner, repo
  "http://github.com/%s/%s" % [owner, repo]
end

redis = Redic.new

repos = Github::Trending.get
repos.each do |r|
  owner, repo = r.name.split("/")
  redis.call "SADD", "hug:bored-urls", canonicalize(owner, repo)
end
