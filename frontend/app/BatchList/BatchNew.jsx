import React from 'react'
import PropTypes from 'prop-types';
import MenuBanner from "../MenuBanner.jsx";
import JobTemplateSelector from "../QuickTranscode/JobTemplateSelector.jsx";
import BasicUploadComponent from "../QuickTranscode/BasicUploadComponent.jsx";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {Redirect} from "react-router-dom"

class BatchNew extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            uploading: false,
            templateId: "",
            createdBatchId: null
        };

        this.fileReadCompleted = this.fileReadCompleted.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()));
    }

    async fileReadCompleted(data) {
        console.log("file ready to upload: ", data);
        const response = await fetch("/api/bulk/upload", {method:"POST", body: data, headers:{"Content-Type":"application/text"}});


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
                <div className="inline-dialog-content" style={{marginTop: "1em"}}>
                    <span className="transcode-info-block" style={{marginBottom: "1em", display:"block"}}>
                        <JobTemplateSelector value={this.state.templateId} onChange={evt=>this.setState({templateId: evt.target.value})}/>
                    </span>
                    <BasicUploadComponent id="upload-box"
                                          loadStart={(file)=>this.setState({uploading: true})}
                                          loadCompleted={this.fileReadCompleted}/>
                    <label htmlFor="upload-box"><FontAwesomeIcon icon="upload" style={{marginRight: "4px"}}/>Upload a list of filenames</label>

                        <span className="error-text" style={{display: this.state.lastError ? "block" : "none"}}>{this.state.lastError}</span>
                    </div>
            </div>
        </div>
    }
}

export default BatchNew;