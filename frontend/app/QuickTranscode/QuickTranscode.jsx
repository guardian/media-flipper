import React from 'react';
import MenuBanner from "../MenuBanner.jsx";
import BasicUploadComponent from "./BasicUploadComponent.jsx";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import BytesFormatterImplementation from "../Common/BytesFormatterImplementation.jsx";
import css from "../inline-dialog.css";

class QuickTranscode extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            phase: 0,
            uploadCompleted: false,
            lastError: null,
            jobId: null,
            fileName: null,
            settingsId: "63AD5DFB-F6F6-4C75-9F54-821D56458279"  //FAKE value for testing
        };

        this.setStatePromise = this.setStatePromise.bind(this);
        this.newDataAvailable = this.newDataAvailable.bind(this);

    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()).catch(err=>reject(err)));
    }

    /**
     * asks the server to create a new job.
     * @returns {Promise<string>} Resolves to the created job ID if successful or rejects otherwise
     */
    async createJobEntry() {
        //see webapp/jobs/jobrequest.go
        const requestContent = JSON.stringify({
            settingsId: this.state.settingsId
        });

        const response = await fetch("/api/job/new",{method:"POST",body: requestContent,headers:{"Content-Type":"application/json"}});
        const responseBody = await response.text();

        if(response.status<200 || response.status>299){
            console.error("Server responded", response.status, response.statusText);
            console.error(responseBody);
            throw "Server error: " + response.statusText;
        }

        const responseJson = JSON.parse(responseBody);
        return responseJson.jobId;
    }

    /**
     * uploads a data buffer to the given job id
     * @param jobId
     * @param data
     * @returns {Promise<void>}
     */
    async uploadData(jobId, data) {
        console.log("uploadData: ", data);
        const response = await fetch("/api/flip/upload?forJob="+jobId, {method:"POST", body: data, headers:{"Content-Type":"application/octet-stream"}});
        const responseBody = await response.text();

        if(response.status<200 || response.status>299){
            console.error("Server responded", response.status, response.statusText);
            console.error(responseBody);
            throw "Server error: " + response.statusText;
        }
    }

    async uploadProcess(data) {
        const newJobId = await this.createJobEntry();
        console.log("Job created with id ",newJobId);
        await this.setStatePromise({jobId: newJobId, phase: 1, uploadCompleted: false});
        await this.uploadData(newJobId, data);
    }

    newDataAvailable(data) {
        this.uploadProcess(data)
            .then(()=>this.setState({phase: 2, uploadCompleted: true}))
            .catch(err=>this.setState({loading: false, lastError: err}))
    }

    render() {
        return <div>
            <MenuBanner/>
            <div className="inline-dialog">
                <h2 className="inline-dialog-title">Quick transcode</h2>
                <div className="inline-dialog-content" style={{marginTop: "1em"}}>
                <BasicUploadComponent id="upload-box"
                                      loadStart={(file)=>this.setState({loading: true, fileName: file.name + " (" + BytesFormatterImplementation.getString(file.size) + " " + file.type + ")"})}
                                      loadCompleted={this.newDataAvailable}/>
                <label htmlFor="upload-box"><FontAwesomeIcon icon="upload" style={{marginRight: "4px"}}/>Upload a file</label>
                <div id="placeholder" style={{height: "4em", display: "block", overflow: "hidden"}}>
                    <span className="transcode-info-block" style={{display: this.state.fileName ? "inherit" : "none"}}>{this.state.fileName}</span>

                    <span className="transcode-info-block" style={{display: this.state.phase<1 ? "none" : "block"}}>{
                        this.state.uploadCompleted ? "Uploading... Done!" : "Uploading..."
                    }</span>
                    <span className="transcode-info-block" style={{display: this.state.phase<2 ? "none" : "block"}}>{
                        this.state.analysisCompleted ? "Analysing... Done!" : "Analysing..."
                    }</span>
                    <span className="error-text" style={{display: this.state.lastError ? "block" : "none"}}>{this.state.lastError}</span>
                </div>
                </div>
            </div>
        </div>
    }
}

export default QuickTranscode;