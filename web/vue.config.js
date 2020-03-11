/*const BundleAnalyzerPlugin = require('webpack-bundle-analyzer')
    .BundleAnalyzerPlugin;*/

module.exports = {
  publicPath: '/static/',
  productionSourceMap: true,
  devServer: {
    host: '0.0.0.0',
    port: 8081,
    disableHostCheck: true /* 0.0.0.0 */
  }
  /*configureWebpack: {
      plugins: [new BundleAnalyzerPlugin({
        analyzerHost: "0.0.0.0",
        analyzerPort: 8081
      })]
  }*/
}
