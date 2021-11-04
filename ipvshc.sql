######################################################################

CREATE TABLE config (
   id         integer not null primary key autoincrement,
   host       text     not null,
   thold      integer  not null,
   interval   integer  not null,
   tgtoken    text    not null,
   descr      text DEFAULT '');

CREATE TABLE healthcheck (
    id        integer not null primary key autoincrement,
    vs        text    not null,
    vaddr     text    not null,
    raddr     text    not null,
    caddr     text    not null,
    path      text    not null,
    mode      text    not null,
    weight    text    not null,
    tgid      text    not null,
    state     text DEFAULT 'enable',
    descr     text DEFAULT '');

CREATE TABLE state (
    id        integer not null primary key autoincrement,
    raddr     text    not null,
    status    text    not null,
    changed   DATETIME DEFAULT CURRENT_TIMESTAMP);


######################################################################

INSERT INTO config (host, thold, interval, tgtoken) VALUES ('10.10.1.1', '3', '10', '12345:AAABBBccc');
INSERT INTO healthcheck (vs, vaddr, raddr, caddr, path, mode, weight, tgid, descr) values('t','10.10.10.1:443', '10.10.10.11:443', '10.10.10.11:8182', 'path','m','5','-11111', 'test' ); INSERT INTO state (raddr, status) VALUES ('10.10.10.11:443', 'PENDING');
INSERT INTO healthcheck (vs, vaddr, raddr, caddr, path, mode, weight, tgid, descr) values('t','10.10.10.1:443', '10.10.10.12:443', '10.10.10.12:8182', 'path','m','5','-11111', 'test' ); INSERT INTO state (raddr, status) VALUES ('10.10.10.12:443', 'PENDING');
INSERT INTO healthcheck (vs, vaddr, raddr, caddr, path, mode, weight, tgid, descr) values('t','10.10.10.1:443', '10.10.10.13:443', '10.10.10.13:8182', 'path','m','5','-11111', 'test' ); INSERT INTO state (raddr, status) VALUES ('10.10.10.13:443', 'PENDING');

######################################################################
