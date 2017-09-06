module.exports = {
  devtool: 'source-map',
  entry: './index',
  output: {
    filename: 'bundle.js',
  },
  resolve: {
    extensions: ['', '.js'],
  },
  module: {
    loaders: [
      {
        test: /\.js$/,
        loader: 'babel?stage=0',
      },
    ],
  },
  node: {
    console: false,
    process: false,
    setImmediate: false,
  },
};
