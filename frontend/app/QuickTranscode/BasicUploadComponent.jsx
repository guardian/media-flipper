import React from 'react';
import PropTypes from 'prop-types';
import css from "./UploadComponent.css";

/**
 * simple component to load the file into browser environment and make it available for upload
 */
class BasicUploadComponent extends React.Component {
    static propTypes = {
        id: PropTypes.string,
        loadCompleted: PropTypes.func.isRequired,
        loadStart: PropTypes.func
    };

    constructor(props) {
        super(props);

        this.fileReader = new FileReader();
        this.fileInputChanged = this.fileInputChanged.bind(this);
        this.fileLoadCompleted = this.fileLoadCompleted.bind(this);
    }

    fileLoadCompleted(evt) {
        console.log("fileLoadCompleted: ", evt);
        if(evt.result===null){
            console.error("fileLoadCompleted but no result data");
            return;
        }
        this.props.loadCompleted(this.fileReader.result);
    }

    fileInputChanged(evt) {
        const file = evt.target.files[0];
        console.log("fileInputChanged: ", file);
        if(!file){
            console.error("fileInputChanged but got no files??");
            return;
        }
        this.fileReader.onloadend = this.fileLoadCompleted;

        if(this.props.loadStart) this.props.loadStart(file);
        console.log("started reading");
        this.fileReader.readAsArrayBuffer(file);
    }

    render(){
        return <input type="file" id={this.props.id} className="inputfile" onChange={this.fileInputChanged}/>
    }
}

export default BasicUploadComponent;