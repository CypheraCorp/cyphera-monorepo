module.exports = {
  apps: [
    {
      name: 'postgres',
      script: 'docker-compose',
      args: 'up postgres',
      interpreter: 'none',
      autorestart: false,
    },
    {
      name: 'delegation-server',
      script: 'npm',
      args: 'run dev:delegation',
      cwd: './',
      env: {
        NODE_ENV: 'development',
      },
    },
    {
      name: 'api-server',
      script: 'npm',
      args: 'run dev:api',
      cwd: './',
      env: {
        NODE_ENV: 'development',
      },
    },
    {
      name: 'web-app',
      script: 'npm',
      args: 'run dev:web',
      cwd: './',
      env: {
        NODE_ENV: 'development',
      },
    },
  ],
};