var webpack = require('webpack');
var path = require('path');
var TerserPlugin = require('terser-webpack-plugin');
var ExtractTextPlugin = require('extract-text-webpack-plugin');

var BUILD_DIR = path.resolve(__dirname + "/..", 'webapp/static');
var APP_DIR = path.resolve(__dirname, 'app');

var config = {
    entry: APP_DIR + '/index.jsx',
    output: {
        path: BUILD_DIR,
        filename: 'bundle.js'
    },
    optimization: {
        minimizer: [new TerserPlugin()]
    },
    module : {
        rules : [
            {
                test : /\.jsx?/,
                include : APP_DIR,
                loader : 'babel-loader'
            },
            {
                test: /\.css$/,
                //use: ['style-loader', 'css-loader']
                //see https://stackoverflow.com/questions/43567527/webpack-bundle-required-css-files
                loader: ExtractTextPlugin.extract("css")
            }
        ]
    },
    plugins: [
        new ExtractTextPlugin(BUILD_DIR + "styles.css")
    ]
};

module.exports = config;