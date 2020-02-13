import React from 'react';
import {render} from 'react-dom';
import {BrowserRouter, Link, Route, Switch, Redirect, withRouter} from 'react-router-dom';
import { library } from '@fortawesome/fontawesome-svg-core'
import RootComponent from "./RootComponent.jsx";
import css from './approot.css';
import JobList from "./JobList/JobList.jsx";
import QuickTranscode from "./QuickTranscode/QuickTranscode.jsx";
import { faUpload, faCloudUploadAlt, faFileUpload, faCaretRight, faCaretDown, faWrench, faTools,
    faCheckCircle, faTimesCircle,faPauseCircle, faPlayCircle, faLayerGroup, faTrash, faFileExport, faHdd, faFolder, faExclamation } from '@fortawesome/free-solid-svg-icons'
import BatchListMain from "./BatchList/BatchListMain.jsx";
import BatchEdit from "./BatchList/BatchEdit.jsx";
import BatchNew  from "./BatchList/BatchNew.jsx";
//import { faPauseCircle } from "@fortawesome/free-regular-svg-icons";

library.add(faUpload, faCloudUploadAlt, faFileUpload, faCaretRight, faCaretDown,faWrench, faTools,
    faCheckCircle, faTimesCircle, faPauseCircle, faPlayCircle, faLayerGroup, faTrash, faFileExport, faHdd, faFolder, faExclamation);

class App extends React.Component {
    render() {
        return <Switch>
                <Route path="/jobs" component={JobList}/>
                <Route path="/quicktranscode" component={QuickTranscode}/>
                <Route path="/batch/new" component={BatchNew}/>
                <Route path="/batch/:batchId" component={BatchEdit}/>
                <Route path="/batch" exact={true} component={BatchListMain}/>
                <Route path="/" exact={true} component={RootComponent}/>
            </Switch>;
    }
}

const AppWithRouter = withRouter(App);

render(<BrowserRouter root="/"><AppWithRouter/></BrowserRouter>, document.getElementById('app'));

export default App;