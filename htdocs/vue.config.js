const { InjectManifest } = require('workbox-webpack-plugin')

module.exports = {
  configureWebpack: {
    plugins: [
      new InjectManifest({
        swSrc: './src/service-worker.js'
      })
    ]
  },

  pluginOptions: {
    i18n: {
      locale: 'en',
      fallbackLocale: 'en',
      localeDir: 'locales',
      enableInSFC: true
    }
  }
}
