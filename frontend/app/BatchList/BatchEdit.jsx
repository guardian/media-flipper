import React from 'react';
import PropTypes from 'prop-types';
import ndjsonStream from "can-ndjson-stream";
import MenuBanner from "../MenuBanner.jsx";
import BatchEntry from "./BatchEntry.jsx";
import css from "../gridform.css";
import appcss from "../approot.css";
import Modal from 'react-modal';
import moment from 'moment';
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import BatchStatusSummary from "./BatchStatusSummary.jsx";
import JobTemplateSelector from "../QuickTranscode/JobTemplateSelector.jsx";

Modal.setAppElement("#app");

class BatchEdit extends React.Component {
    constructor(props){
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            batchId: "",
            batchCreated: "",
            batchNickname: "",
            videoTemplateId: "00000000-0000-0000-0000-000000000000",
            audioTemplateId: "00000000-0000-0000-0000-000000000000",
            imageTemplateId: "00000000-0000-0000-0000-000000000000",
            pendingCount: 0,
            activeCount: 0,
            completedCount: 0,
            errorCount: 0,
            entries: [],
            currentReader: null,
            currentAbort: null,
            pageLoadLimit: 30,
            pageItemsLimit: 1000,
            itemsInPage: 0,
            templateEntries: [],
            scrollPosition: 0,
            showingModalWarning: false,
            jobUpdateTimer: null,
            statusFilter: null
        };

        this.triggerRemoveDotFiles = this.triggerRemoveDotFiles.bind(this);
        this.triggerRemoveNonTranscodable = this.triggerRemoveNonTranscodable.bind(this);
        this.maybeTriggerNonTranscodable = this.maybeTriggerNonTranscodable.bind(this);
        this.triggerEnqueueItems = this.triggerEnqueueItems.bind(this);
        this.retryRequested = this.retryRequested.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async retryRequested(forId) {
        console.log("Retry requested for ", forId);
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/jobrunner/enqueue?forId=" + this.state.batchId + "&forItem=" + forId, {method:"POST"});
        if(response.status===200){
            await response.body.cancel();
            return this.loadExistingData(this.state.batchId);
        } else {
            const bodyContent = await response.text();
            return this.setStatePromise({loading: false, lastError: bodyContent})
        }
    }

    async triggerRemoveDotFiles() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/action/removeDotFiles?forId=" + this.state.batchId, {method: "POST"});
        if(response.status===200) {
            await response.body.cancel();
            await this.loadExistingData(this.state.batchId);
        } else {
            const bodyContent = await response.text();
            await this.setStatePromise({loading: false, lastError: bodyContent});
        }
    }

    maybeTriggerNonTranscodable() {
        if(this.state.videoTemplateId==="00000000-0000-0000-0000-000000000000" ||
            this.state.audioTemplateId==="00000000-0000-0000-0000-000000000000" ||
            this.state.imageTemplateId==="00000000-0000-0000-0000-000000000000"
        ){  //show a warning if any settings are unset
            this.setState({showingModalWarning: true});
        } else {
            this.triggerRemoveNonTranscodable();
        }
    }

    async triggerRemoveNonTranscodable() {
        await this.setStatePromise({loading: true, showingModalWarning: false});
        const response = await fetch("/api/bulk/action/removeNonTranscodable?forId=" + this.state.batchId, {method: "POST"});
        if(response.status===200) {
            await response.body.cancel();
            await this.loadExistingData(this.state.batchId);
        } else {
            const bodyContent = await response.text();
            await this.setStatePromise({loading: false, lastError: bodyContent});
        }
    }

    async triggerEnqueueItems() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/jobrunner/enqueue?forId=" + this.state.batchId, {method:"POST"});
        if(response.status===200){
            await response.body.cancel();
            return this.loadExistingData(this.state.batchId);
        } else {
            const bodyContent = await response.text();
            return this.setStatePromise({loading: false, lastError: bodyContent})
        }
    }

    async storeUpdatedInfo() {
        await this.setStatePromise({loading: true});
        const request = JSON.stringify({"nickName": this.state.batchNickname,
            "videoTemplateId": this.state.videoTemplateId,
            audioTemplateId: this.state.audioTemplateId,
            imageTemplateId: this.state.imageTemplateId
        });
        const response = await fetch("/api/bulk/update?forId=" + this.state.batchId, {
            body: request,
            method: "POST",
            headers: {"Content-Type": "application/json" },
        });

        if(response.status===200) {
            await response.body.cancel(); //don't actually care about the content
            console.log("update saved");
            return this.setStatePromise({loading: false, lastError: null});
        } else {
            const responseText = response.text();
            try {
                const content = JSON.parse(responseText);
                return this.setStatePromise({loading: false, lastError: content.detail});
            } catch(e) {
                return this.setStatePromise({loading: false, lastError: responseText});
            }
        }
    }

    async componentDidUpdate(prevProps, prevState, snapshot) {
        if(this.state.batchNickname!==prevState.batchNickname || this.state.videoTemplateId!==prevState.videoTemplateId || this.state.audioTemplateId !== prevState.audioTemplateId || this.state.imageTemplateId !== prevState.imageTemplateId) {
            await this.storeUpdatedInfo();
        }
        if(this.state.statusFilter !== prevState.statusFilter) {
            console.log("statusFilter updated");
            const batchId = this.props.match.params.batchId;
            await this.loadBatchContent(batchId);
        }
    }

    async loadTemplatesList() {
        await this.setStatePromise({loading: true, lastError: null});

        const response = await fetch("/api/jobtemplate");
        if(response.status===200){
            const serverData = await response.json();
            const templateEntries = serverData.entries;
            const templateIdUpdate = templateEntries.length>0 ? {templateId: templateEntries[0].Id} : {};

            await this.setStatePromise(Object.assign({loading: false, lastError: null, templateEntries: templateEntries}, templateIdUpdate));
        } else {
            const bodyText = await response.text();

            return this.setStatePromise({loading: false, lastError: bodyText})
        }
    }

    async loadExistingData(batchId) {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/get?forId=" + batchId);
        if(response.status===200){
            const content = await response.json();
            return this.setStatePromise({
                loading: false,
                lastError: null,
                batchId: content.bulkListId,
                batchNickname: content.nickName,
                videoTemplateId: content.videoTemplateId,
                audioTemplateId: content.audioTemplateId,
                imageTemplateId: content.imageTemplateId,
                batchCreated: content.creationTime,
                pendingCount: content.pendingCount,
                activeCount: content.activeCount,
                completedCount: content.completedCount,
                errorCount: content.errorCount,
                abortedCount: content.abortedCount,
                nonQueuedCount: content.nonQueuedCount,
                removeFilesRunning: content.runningActions.includes("remove-system-files"),
                enqueueRunning: content.runningActions.includes("jobs-queueing"),
            })
        } else {
            try {
                const jsonContent = await response.json();
                return this.setStatePromise({loading: false, lastError: jsonContent.detail});
            } catch (e) {
                console.error("could not get json error response from server: ", e);
                const bodyContent = await response.text();
                return this.setStatePromise({loading: false, lastError: bodyContent})
            }
        }
    }

    async loadBatchContent(batchId) {
        if(this.state.currentAbort!==null) {
            console.log("aborting current operation....");
            this.state.currentAbort.abort();
        }
        await this.setStatePromise({loading: true, entries:[], currentAbort: null});

        const abortController = new AbortController();

        let baseUrl = "/api/bulk/content?";
        if(this.state.statusFilter) {
            baseUrl = "/api/bulk/content?state=" + this.state.statusFilter + "&"
        }
        console.log("baseUrl is ", baseUrl, " statusFilter is ", this.state.statusFilter);

        const response = await fetch(baseUrl + "forId=" + batchId, {signal: abortController.signal});
        const stream = await ndjsonStream(response.body);
        const reader = stream.getReader();

        await this.setStatePromise({currentReader: reader, currentAbort: abortController});

        function readNextChunk(reader, currentCount) {
            reader.read().then(({done, value}) =>{
                if(value) {
                    this.setState(oldState=>{
                        return {entries: oldState.entries.concat([value]), itemsInPage: oldState.items +1}
                    }, ()=>{
                        if(done || currentCount>=this.state.pageLoadLimit-1) {
                            this.setState({loading: false, lastError: null, currentAbort: null});
                        } else {
                            window.setTimeout(()=> {
                                readNextChunk(reader, currentCount + 1);
                            },25); //delay each one by 1/40 second to let renderer catch up
                        }
                    })
                } else {
                    console.warn("Got no data");
                }
            })
        }
        readNextChunk = readNextChunk.bind(this);
        readNextChunk(reader,0);
    }

    // shouldComponentUpdate(nextProps, nextState, nextContext) {
    //     if(nextState.entriesBackingStore!==this.state.entriesBackingStore){
    //         return nextState.entries.length-nextState.;
    //     }
    //     return true;
    // }

    async componentDidMount() {
        window.addEventListener('scroll', this.listenToScroll);
        const batchId = this.props.match.params.batchId;
        await this.loadTemplatesList();

        if(batchId!=="new" && batchId!=="") {
            await this.loadExistingData(batchId);
            await this.setStatePromise({jobUpdateTimer: window.setInterval(()=>this.loadExistingData(batchId), 3000)});

            return this.loadBatchContent(batchId);
        }
    }

    componentWillUnmount() {
        window.removeEventListener('scroll', this.listenToScroll);
        if(this.state.jobUpdateTimer) {
            window.clearTimeout(this.state.jobUpdateTimer);
        }
    }

    listenToScroll = () => {
        const winScroll =
            document.body.scrollTop || document.documentElement.scrollTop;

        const height =
            document.documentElement.scrollHeight -
            document.documentElement.clientHeight;

        const scrolled = winScroll / height;

        this.setState({
            scrollPosition: scrolled,
        })
    };

    render() {
        return <div>
            <MenuBanner/>
            <h2>Edit Batch</h2>
            <div className="grid-form">
                <label className="grid-form-label" htmlFor="nickname">Friendly name</label>
                <input className="grid-form-control-stretch" id="nickname" value={this.state.batchNickname} onChange={evt=>this.setState({batchNickname: evt.target.value})}/>
                <label className="grid-form-label" htmlFor="created">Created at</label>
                <span id="created" className="grid-form-control">{moment(this.state.batchCreated).format("ddd, MMM Do YYYY, h:mm:ss a")}</span>

                <label className="grid-form-label" htmlFor="video-preset-id">Video transcoding preset</label>
                <div className="grid-form-control">
                    <JobTemplateSelector jobTemplateList={this.state.templateEntries} onChange={evt=>this.setState({videoTemplateId: evt.target.value})} value={this.state.videoTemplateId}/>
                </div>
                <div style={{display: this.state.videoTemplateId==="00000000-0000-0000-0000-000000000000" ? "initial" : "none", fontWeight: "bold", fontSize: "0.9em"}} className="grid-form-indicator"><FontAwesomeIcon icon="exclamation" style={{color: "orange"}}/>&nbsp;You should select a template</div>

                <label className="grid-form-label" htmlFor="video-preset-id">Audio transcoding preset</label>
                <div className="grid-form-control">
                    <JobTemplateSelector jobTemplateList={this.state.templateEntries} onChange={evt=>this.setState({audioTemplateId: evt.target.value})} value={this.state.audioTemplateId}/>
                </div>
                <div style={{display: this.state.audioTemplateId==="00000000-0000-0000-0000-000000000000" ? "initial" : "none", fontWeight: "bold", fontSize: "0.9em"}} className="grid-form-indicator"><FontAwesomeIcon icon="exclamation" style={{color: "orange"}}/>&nbsp;You should select a template</div>

                <label className="grid-form-label" htmlFor="video-preset-id">Image transcoding preset</label>
                <div className="grid-form-control">
                    <JobTemplateSelector jobTemplateList={this.state.templateEntries} onChange={evt=>this.setState({imageTemplateId: evt.target.value})} value={this.state.imageTemplateId}/>
                </div>
                <div style={{display: this.state.imageTemplateId==="00000000-0000-0000-0000-000000000000" ? "initial" : "none", fontWeight: "bold", fontSize: "0.9em"}} className="grid-form-indicator"><FontAwesomeIcon icon="exclamation" style={{color: "orange"}}/>&nbsp;You should select a template</div>


                <label className="grid-form-label" htmlFor="status">Status</label>
                <BatchStatusSummary batchStatus={this.state}
                                    className="grid-form-control-stretch"
                                    filterClicked={(newValue)=>this.setState({statusFilter: newValue})}
                                    currentFilterName={this.state.statusFilter}
                />

                <label className="grid-form-label" htmlFor="progress">Overall progress</label>
                <span id="progress" className="grid-form-control emphasis">{Math.ceil(100*(this.state.completedCount+this.state.errorCount)/(this.state.pendingCount+this.state.activeCount + this.state.completedCount + this.state.errorCount))} %</span>

                <label className="grid-form-label" htmlFor="actions">Actions</label>
                <span id="actions" className="grid-form-control-stretch">
                    <ul className="status-summary-container">
                        <li className={this.state.removeFilesRunning ? "status-summary-entry button disabled"  : "status-summary-entry button clickable"}
                            onClick={this.triggerRemoveDotFiles} style={{marginRight:"2em"}}>
                            <FontAwesomeIcon icon="minus-circle" style={{padding: "0.4em"}}/>Remove system files
                        </li>
                        <li className="status-summary-entry button clickable" onClick={this.maybeTriggerNonTranscodable} style={{marginRight:"2em"}}>
                            <FontAwesomeIcon icon="minus-circle"  style={{padding: "0.4em"}}/>Remove non-transcodable files
                        </li>
                        <li className="status-summary-entry button clickable" onClick={this.triggerEnqueueItems} style={{marginRight:"2em"}}>
                            <FontAwesomeIcon icon="play-circle"  style={{padding: "0.4em"}}/>Start jobs running
                        </li>
                    </ul>
                </span>
            </div>
            <ul className="batch-list">
                {
                    this.state.entries.map(ent=><li key={ent.id}>
                        <BatchEntry entry={ent}
                                    validAudioSettings={this.state.audioTemplateId!=="00000000-0000-0000-0000-000000000000"}
                                    validVideoSettings={this.state.videoTemplateId!=="00000000-0000-0000-0000-000000000000"}
                                    validImageSettings={this.state.imageTemplateId!=="00000000-0000-0000-0000-000000000000"}
                                    onRetryRequested={this.retryRequested}
                        /></li>)
                }
            </ul>

            <Modal isOpen={this.state.showingModalWarning}
                   onRequestClose={()=>this.setState({showingModalWarning: false})}
                   style={{content: {maxHeight: "300px", maxWidth:"600px", marginLeft:"auto",marginRight:"auto"}}}
            >
                <h2>Are you sure?</h2>
                <p className="warning">You have not selected templates for all of the media types. If you continue, then any items
                    of types that don't have a template will be removed. Are you sure you want to continue?</p>
                <div style={{maxWidth: "400px", marginRight: "auto", marginLeft: "auto"}}>
                <label className="button clickable" onClick={this.triggerRemoveNonTranscodable}>Continue</label>
                <label className="button clickable" onClick={()=>this.setState({showingModalWarning: false})}>Cancel</label>
                </div>
            </Modal>
        </div>;
    }
}

export default BatchEdit;