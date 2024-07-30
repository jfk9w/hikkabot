create index if not exists event_reddit_user_id_idx on event (chat_id, (data -> 'user_id'))
    where ((data -> 'user_id')) is not null;

create index if not exists event_reddit_user_id_message_id_idx on event (chat_id, (data -> 'user_id'), (data -> 'message_id'))
    where ((data -> 'user_id')) is not null and (data -> 'message_id') is not null;

create index if not exists event_reddit_subreddit_idx on event (chat_id, (data ->> 'subreddit'))
    where ((data ->> 'subreddit')) is not null;

create index if not exists event_reddit_thing_id_idx on event (chat_id, (data ->> 'thing_id'))
    where ((data ->> 'thing_id')) is not null;

create index if not exists event_reddit_user_id_thing_id_idx on event (chat_id, (data -> 'user_id'), (data ->> 'thing_id'))
    where ((data -> 'user_id')) is not null and (data ->> 'thing_id') is not null;