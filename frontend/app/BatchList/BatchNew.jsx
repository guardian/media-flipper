import React from 'react'
import PropTypes from 'prop-types';
import MenuBanner from "../MenuBanner.jsx";
import JobTemplateSelector from "../QuickTranscode/JobTemplateSelector.jsx";
import BasicUploadComponent from "../QuickTranscode/BasicUploadComponent.jsx";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {Redirect} from "react-router-dom"
import BytesFormatterImplementation from "../Common/BytesFormatterImplementation.jsx";

class BatchNew extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            uploading: false,
            reading: false,
            fileName: "",
            readingProgress: 0.0,
            templateId: "",
            createdBatchId: null
        };

        this.fileReadCompleted = this.fileReadCompleted.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()));
    }

    async fileReadCompleted(data) {
        await this.setStatePromise({uploading: true, readingProgress: 1.0});

        console.log("file ready to upload: ", data);
        const response = await fetch("/api/bulk/upload", {method:"POST", body: data, headers:{"Content-Type":"application/text; charset=ISO-8859-1"}});


        if(response.status<200 || response.status>299){
            const responseBody = await response.text();
            console.error("Server responded", response.status, response.statusText);
            console.error(responseBody);
            throw "Server error: " + response.statusText;
        } else {
            const responseContent = await response.json();
            return this.setStatePromise({
                uploading: false,
                createdBatchId: responseContent.bulkid
            })
        }
    }

    render() {
        if(this.state.createdBatchId) return <Redirect to={"/batch/" + this.state.createdBatchId}/>;

        return <div>
            <MenuBanner/>
            <div className="inline-dialog">
                <h2 className="inline-dialog-title">New Batch</h2>
                <div className="inline-dialog-content" style={{marginTop: "1em", overflow:"hidden"}}>
                    <BasicUploadComponent id="upload-box"
                                          loadStart={(file)=>this.setState({reading: true, fileName: file.name + " (" + BytesFormatterImplementation.getString(file.size) + " " + file.type + ")"})}
                                          loadCompleted={this.fileReadCompleted}
                                          loadProgress={(pct)=>this.setState({readingProgress: pct})}
                    />
                    <label htmlFor="upload-box" className="clickable" style={{display: (this.state.reading || this.state.uploading )? "none" : "inherit"}}><FontAwesomeIcon icon="upload" style={{marginRight: "4px"}}/>Upload a list of filenames</label>

                    <ul style={{listStyle: "none"}}>
                    <li className="error-text" style={{display: this.state.lastError ? "block" : "none"}}>{this.state.lastError}</li>
                        <li className="transcode-info-block" style={{display: this.state.fileName==="" ? "none" : "inherit"}}>{this.state.fileName}</li>
                    <li className="transcode-info-block" style={{display: this.state.reading ? "inherit" : "none"}}>Reading, {this.state.readingProgress*100.0}%...</li>
                        <li className="transcode-info-block" style={{display: this.state.uploading ? "inherit" : "none"}}>Uploading...</li>
                    </ul>
                    </div>
            </div>
        </div>
    }
}

export default BatchNew;