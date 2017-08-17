var cluster = require('cluster');
var express = require('express');
var app = express();
var bodyParser = require('body-parser');
var child_process = require('child_process');
var fs = require('fs');
var os = require('os');
var crypto = require('crypto');

var available_containers = [];

var count_container = 3;

var md5hex = function(src){
	var md5hash = crypto.createHash('md5');
	md5hash.update(src, 'binary');
	return md5hash.digest('hex');
}

function create_container(){
	var runningHash = md5hex(new Date().getTime().toString());
	var workspace = '/tmp/workspace/' + runningHash + "/";
	var dockerCmd = 
		'docker run -i -d ' + 
		'--net none ' + 
		'--memory 512m --memory-swap 512m ' +
		'--ulimit nproc=10:10 ' +
		'--ulimit fsize=1000000 ' +
		'-w /workspace/' + runningHash + ' ' +  
		'ugwis/online-compiler ' +
		'/bin/ash';
	console.log("Running: " + dockerCmd);
	var output = child_process.execSync(dockerCmd);
	console.log(output.toString());
	var containerId = output.toString().substr(0,12);
	console.log(containerId);
	console.log("ok");
	available_containers.push({
		'runningHash': runningHash,
		'containerId': containerId,
		'workspace': workspace
	});
}


var numCPUs = os.cpus().length;
if(cluster.isMaster){
	for(var i=0; i < numCPUs; i++){
		cluster.fork();
	}
} else {
	for(var i=0;i < count_container;i++) {
		create_container();
	}
	console.log(available_containers);
	var languages = {
		'ruby': {
			filename: 'Main.rb',
			execCmd: 'ruby Main.rb'
		},
		'python': {
			filename: 'Main.py',
			execCmd: 'python Main.py'
		},
		'c': {
			filename: 'Main.c',
			execCmd: 'gcc -Wall -o Main Main.c && ./Main'
		},
		'cpp': {
			filename: 'Main.cpp',
			execCmd: 'g++ -Wall -o Main Main.cpp && ./Main'
		},
		'cpp11': {
			filename: 'Main.cpp',
			execCmd: 'g++ -std=c++0x -o Main Main.cpp && ./Main'
		},
		'php': {
			filename: 'Main.php',
			execCmd: 'php Main.php'
		},
		'js': {
			filename: 'Main.js',
			execCmd: 'node Main.js'
		},
		'bash': {
			filename: 'Main.sh',
			execCmd: 'bash Main.sh'
		}
	}

	app.use(express.static('public'));
	app.use(bodyParser.urlencoded({extended: false}));

	app.post('/api/run', function(req, res){
		res.setHeader("Access-Control-Allow-Origin", "*");
		var language = req.body.language===undefined?"":req.body.language;
		var source_code = req.body.source_code===undefined?"":req.body.source_code;
		var input = req.body.input===undefined?"":req.body.input;

		var runningHash = md5hex(new Date().getTime().toString());
		var workspace = '/tmp/workspace/' + runningHash + "/";

		var filename, execCmd;

		// Chose container 
		while(available_containers.length == 0) sleep(0.001);
		var container = available_containers.pop();
		var containerId = container.containerId;
		var runningHash = container.runningHash;
		var workspace = container.workspace;
		var compileHash = md5hex(new Date().getTime().toString());
	
		// Copy the source code to the container
		child_process.execSync('mkdir -p ' + workspace + ' && chmod 777 ' + workspace + '/');
		fs.writeFileSync(workspace + languages[language].filename, source_code);
		dockerCmd = "docker cp " + workspace + ' ' + containerId + ":/workspace/";
		console.log("Running: " + dockerCmd);
		child_process.execSync(dockerCmd);
		console.log("ok");

		/*
		dockerCmd = "docker exec -i " + containerId + " ls"; 
		console.log(dockerCmd);
		console.log(child_process.execSync(dockerCmd).toString());
			*/	

		// Start compile
		
		dockerCmd = 'docker exec -i ' + containerId + ' timeout -t 3 ' +
			'su nobody -s /bin/ash -c "' +
			languages[language].execCmd +
			'"'; 
		//dockerCmd = 'docker exec -i ' + containerId + ' ls';
		console.log("Running: " + dockerCmd);
		var child = child_process.exec(dockerCmd, {}, function(error, stdout, stderr){
			if(error) console.log(error);
			if(stdout) console.log(stdout);
			if(stderr) console.log(stderr);
			res.send({
				stdout: stdout,
				stderr: stderr,
				exit_code: error && error.code || 0,
				/*time: time*/
			});

			console.log("ok");
			//Copy time comand result
			/*dockerCmd = "docker cp " + containerId + ":/time-" + runningHash + ".txt /tmp/time-" + runningHash + ".txt";
			console.log("Running: " + dockerCmd);
			child_process.execSync(dockerCmd);
			var time = fs.readFileSync("/tmp/time-" + runningHash + ".txt").toString();*/
			// Remove the container
			dockerCmd = "docker rm -f " + containerId;
			console.log("Running: " + dockerCmd);
			child_process.execSync(dockerCmd);
			
			console.log("Result: ", error, stdout, stderr);

			child_process.exec('rm -rf ' + workspace,{},function(){});
			create_container();

		});
		child.stdin.write(input)
		child.stdin.end();
	});

	app.listen(3000, function(){
		console.log('Listening on port 3000');
	});
}
