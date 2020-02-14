import React from 'react';
import PropTypes from 'prop-types';
import JobStatusComponent from "../JobList/JobStatusComponent.jsx";
import {FontAwesomeIcon} from '@fortawesome/react-fontawesome';

class BatchEntry extends React.Component {
    static propTypes = {
        entry: PropTypes.object.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            isDotFile: false,
            volumeName: "",
            pathParts: [],
            fileName: ""
        }
    }

    extractState() {
        return new Promise((resolve, reject)=>{
            if(!this.props.entry.sourcePath){
                reject("source path was null");
                return
            }
            let pathParts = this.props.entry.sourcePath.split("/");
            const fileName = pathParts.pop();
            let volumeName;
            if(pathParts[1]==="srv" || pathParts[1]==="Volumes") {
                pathParts.shift();
                pathParts.shift();
                volumeName = pathParts.shift();
            } else {
                volumeName = "(unknown)"
            }
            this.setState({
                isDotFile: fileName[0]===".",
                pathParts: pathParts,
                fileName: fileName,
                volumeName: volumeName
            });

        })
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        if(prevProps.entry!==this.props.entry){
            this.extractState();
        }
    }

    componentDidMount() {
        this.extractState();
    }

    render() {
        const baseClasses = ["batch-entry-cell", "baseline", "item-display-grid"];
        const finalClasses = this.state.isDotFile || (this.props.entry && this.props.entry.type==="other") ? baseClasses.concat(["dot-file"]) : baseClasses;

        return <div className="batch-entry-container">
            <div className={finalClasses.join(" ")}>
                <div className="item-display-element icon"><FontAwesomeIcon icon="file-export"/></div>
                <div className="item-display-element content"><p className="no-spacing emphasis">{this.state.fileName}</p></div>

                <div className="item-display-element icon"><FontAwesomeIcon icon="hdd"/></div>
                <div className="item-display-element content"><p className="no-spacing">{this.state.volumeName}</p></div>

                <div className="item-display-element icon"><FontAwesomeIcon icon="folder"/></div>
                <div className="item-display-element content"><p className="no-spacing small">{ this.state.pathParts.length>0 ? this.state.pathParts.join("/") : ""}</p></div>

                <div className="item-display-element icon" style={{display: this.state.isDotFile ? "inherit": "none"}}><FontAwesomeIcon icon="exclamation" style={{color: "darkorange"}}/></div>
                <div className="item-display-element content" style={{display: this.state.isDotFile ? "inherit": "none"}}><p className="no-spacing small">This is probably a system metadata file and won't transcode</p></div>

                <div className="item-display-element icon" style={{display: !this.state.isDotFile && this.props.entry && this.props.entry.type==="other" ? "inherit": "none"}}>
                    <FontAwesomeIcon icon="exclamation" style={{color: "darkorange"}}/>
                </div>
                <div className="item-display-element content" style={{display: !this.state.isDotFile && this.props.entry && this.props.entry.type==="other"  ? "inherit": "none"}}>
                    <p className="no-spacing small">File extension is unrecognised so we don't know which transcode settings to apply</p>
                </div>

            </div>
            <div className="batch-entry-cell mini"><JobStatusComponent status={this.props.entry.state}/></div>
        </div>
    }
}

export default BatchEntry;