import React from 'react';
import JobStatusComponent from "./JobStatusComponent.jsx";
import MenuBanner from "../MenuBanner.jsx";
import TableView from 'react-table-view';
import MediaFileInfo from "./MediaFileInfo.jsx";

class JobList extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            content: [{
                containerId: "",
                mediaFile: "",
                settingsId: "",
                jobStatus: ""
            }]  //we need a placeholder entry on initial mount so that the table component can find its fields
        };

        this.columns = {
            containerId: (data)=><span className="table-data">{data.containerId}</span>,
            mediaFile: (data)=><div className="table-data">
                <span style={{display: "block"}}>{data.mediaFile.split("/").pop()}</span>
                <MediaFileInfo jobId={data.containerId}/>
            </div>,
            settingsId: (data)=><span className="table-data">{data.settingsId}</span>,
            jobStatus: (data)=><JobStatusComponent className="table-data" status={data.jobStatus}/>
        }
    }

    setStatePromise(newState){
        return new Promise((resolve,reject)=>
            this.setState(newState, ()=>resolve()))
    }

    async loadData(){
        await this.setStatePromise({loading: true});

        try {
            const response = await fetch("/api/job?limit=50");
            const downloadedBody = await response.text();
            if(response.status!==200){
                throw response.status + " error: " + downloadedBody;
            }
            const downloadedData = JSON.parse(downloadedBody);
            return this.setStatePromise({loading: false, lastError: null, content:downloadedData.entries});
        } catch(err){
            console.error("Could not download data: ", err);
            return this.setStatePromise({loading: false, lastError: err});
        }
    };

    componentDidMount() {
        this.loadData();
    }

    render() {
        return <div>
            <MenuBanner/>
            <TableView data={this.state.content} columns={this.columns}/>
        </div>
    }
}

export default JobList;