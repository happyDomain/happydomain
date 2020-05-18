const { InjectManifest } = require('workbox-webpack-plugin')

module.exports = {
  configureWebpack: {
    plugins: [
      new InjectManifest({
        swSrc: './src/service-worker.js'
      })
    ]
  }
}
