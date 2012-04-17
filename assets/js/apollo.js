
var ApolloApp = (function(context){
    var config = null;
    var ws = null;
    var render = null;

    function runApp(cfg) {
        config = cfg
        config.fps = 10;

        render = new Render(config.canvas[0]);
        if (!render.init()) {
            config.noCanvas();
            return
        }

        ws = new WsConn();
        if (!ws.open(config.wsURL)) {
            config.noWebSockets()
            return
        }
        
        render.start(config.fps);

        // TODO really?...
        config.canvas.bind('click', function() {
            var id = null;
            for (id in render.entities) {
                if (!render.entities.hasOwnProperty(id)) { continue; }
                break;
            }
            if (id === null) { return; }
            entityRemove = {
                Act: {
                    G: {
                        C: WsConn.PlayerCmd.GameRemoveEntity,
                        E: parseInt(id)
                    }
                }
            };
            ws.conn.send(JSON.stringify(entityRemove));
        });
    }


    // Webseocket wrapper object
    function WsConn() {
        this.conn = null;
    };
    WsConn.PlayerCmd = {
        GameRemoveEntity: 0
    };
    WsConn.UpdateTypes = {entityRemove: 0, entityAdd: 1};
    WsConn.EntityTypes = {block:0};
    WsConn.prototype.open = function(url) {
        if (window["WebSocket"]) {
            var conn = this.conn = new WebSocket(url);
            var ws = this;
            conn.onclose = function(evt) { ws.onClose(evt); };
            conn.onmessage = function(evt) { ws.onMessage(evt); };
            return true;
        }
        return false
    };
    WsConn.prototype.onClose = function(evt) {
        console.log('Connection Closed,', evt);
        this.conn = null
    };
    WsConn.prototype.onMessage = function(evt) {
        var msg = JSON.parse(evt.data);
        if (msg.BU) { // Board updates
            var updates = msg.BU;
            for (var idx=0; idx < updates.length; idx++) {
                var update = updates[idx]

                if (update.T === WsConn.UpdateTypes.entityAdd) { // Update Type, Add
                    render.addEntity(update.E); 
                    // if (update.E.T === WsConn.EntityTypes.block) { // Entity Type Block
                    // }
                } else if (update.T === WsConn.UpdateTypes.entityRemove) { // Update type, Remove
                    render.removeEntityById(update.E.ID);
                    // if (update.E.T === WsConn.EntityTypes.block) { // Entity type block
                    //     removeBlock(update.E);
                    // }
                }
            }
        }
    };


    // Rendering for drawing 
    function Render(canvas) {
        this.canvas = canvas;
        this.ctx = null;
        this.timer = null;
        this.entities = {};
        this.changed
    };
    Render.prototype.init = function() {
        if (this.canvas.getContext){
            this.ctx = this.canvas.getContext('2d');
            return true
        }
        return false
    };
    Render.prototype.addEntity = function(e) {
        this.entities[e.ID] = e;
        this.changed = true;
    };
    Render.prototype.removeEntityById = function(id) {
        delete this.entities[id];
        this.changed = true;
    };
    Render.prototype.start = function(fps) {
        var interval = 1000 / fps; // calculate frames per second as interval
        var r = this;
        this.timer = setInterval(function() {
            r.draw();
        }, interval);
    };
    Render.prototype.draw = function() {
        if (!this.changed) { return; }
        this.changed = false;

        // It is very inefficent to be clearing the whole canvase each time
        // and we should not be referencing wsConn's statics. they should
        // be building objects we can injest instead of the messages.
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height)
        var _entities = this.entities;
        for (var id in _entities) {
            var e = _entities[id];
            if (e.T === WsConn.EntityTypes.block) {
                this.drawBlock(e);
            }
        }

    };
    Render.prototype.drawBlock = function(block) {
        this.ctx.fillStyle = "rgba("+block.R+","+block.G+","+block.B+",0."+block.A+")";
        this.ctx.fillRect(block.X, block.Y, block.W, block.H);
    }


    return {
        runApp: runApp,
    };
})(this);
