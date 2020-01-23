import React from 'react';
import PropTypes from 'prop-types';

/**
 * simple component to load the file into browser environment and make it available for upload
 */
class BasicUploadComponent extends React.Component {
    static propTypes = {
        id: PropTypes.string,
        className: PropTypes.string,
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
        this.props.loadCompleted(evt.result);
    }

    fileInputChanged(evt) {
        const file = evt.target.files[0];
        if(!file){
            console.error("fileInputChanged but got no files??");
            return;
        }

        if(this.props.loadStart) this.props.loadStart();

        this.fileReader.onloadend = this.fileLoadCompleted;
        this.fileReader.readAsArrayBuffer(file);
    }

    render(){
        return <input type="file" id={this.props.id} className={this.props.className} onChange={this.fileInputChanged}/>
    }
}

export default BasicUploadComponent;