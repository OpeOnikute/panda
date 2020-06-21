db = new Mongo().getDB("admin");

// create admin user
db.createUser({
  user: "admin",
  pwd: "pass",
  roles: [{
    role: "clusterAdmin",
    db: "admin"
  }]
});

db.createUser({
    user: "opeo",
    pwd: "root",
    roles: [
        {
            role: "dbOwner",
            db: "go-panda"
        }
    ]
});